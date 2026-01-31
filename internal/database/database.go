package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	*sql.DB
}

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Domain struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	DocumentRoot string    `json:"document_root"`
	PHPVersion   string    `json:"php_version"`
	SSLEnabled   bool      `json:"ssl_enabled"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Database struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	DomainID  *int64    `json:"domain_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WordPress struct {
	ID        int64     `json:"id"`
	DomainID  int64     `json:"domain_id"`
	Title     string    `json:"title"`
	AdminUser string    `json:"admin_user"`
	DBName    string    `json:"db_name"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type Backup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	DomainID  *int64    `json:"domain_id,omitempty"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// Set connection pool settings for performance
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	d := &DB{db}
	if err := d.migrate(); err != nil {
		return nil, err
	}

	return d, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		email TEXT,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		document_root TEXT NOT NULL,
		php_version TEXT DEFAULT '8.2',
		ssl_enabled INTEGER DEFAULT 0,
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS databases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		username TEXT NOT NULL,
		domain_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (domain_id) REFERENCES domains(id)
	);

	CREATE TABLE IF NOT EXISTS wordpress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain_id INTEGER NOT NULL,
		title TEXT,
		admin_user TEXT,
		db_name TEXT,
		version TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (domain_id) REFERENCES domains(id)
	);

	CREATE TABLE IF NOT EXISTS backups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		domain_id INTEGER,
		file_path TEXT NOT NULL,
		size INTEGER DEFAULT 0,
		status TEXT DEFAULT 'completed',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (domain_id) REFERENCES domains(id)
	);

	CREATE TABLE IF NOT EXISTS ssl_certs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain_id INTEGER NOT NULL,
		cert_path TEXT NOT NULL,
		key_path TEXT NOT NULL,
		provider TEXT DEFAULT 'self-signed',
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (domain_id) REFERENCES domains(id)
	);

	CREATE INDEX IF NOT EXISTS idx_domains_name ON domains(name);
	CREATE INDEX IF NOT EXISTS idx_databases_name ON databases(name);
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) HasAdmin() bool {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	return count > 0
}

func (db *DB) CreateUser(username, password, email, role string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO users (username, password, email, role) VALUES (?, ?, ?, ?)",
		username, string(hash), email, role,
	)
	return err
}

func (db *DB) AuthenticateUser(username, password string) (*User, error) {
	var user User
	var hash string

	err := db.QueryRow(
		"SELECT id, username, password, email, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &hash, &user.Email, &user.Role, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *DB) GetUser(id int64) (*User, error) {
	var user User
	err := db.QueryRow(
		"SELECT id, username, email, role, created_at FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt)
	return &user, err
}

// Domain operations
func (db *DB) CreateDomain(name, docRoot, phpVersion string) (*Domain, error) {
	result, err := db.Exec(
		"INSERT INTO domains (name, document_root, php_version) VALUES (?, ?, ?)",
		name, docRoot, phpVersion,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetDomain(id)
}

func (db *DB) GetDomain(id int64) (*Domain, error) {
	var d Domain
	err := db.QueryRow(
		"SELECT id, name, document_root, php_version, ssl_enabled, active, created_at FROM domains WHERE id = ?", id,
	).Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.PHPVersion, &d.SSLEnabled, &d.Active, &d.CreatedAt)
	return &d, err
}

func (db *DB) GetDomainByName(name string) (*Domain, error) {
	var d Domain
	err := db.QueryRow(
		"SELECT id, name, document_root, php_version, ssl_enabled, active, created_at FROM domains WHERE name = ?", name,
	).Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.PHPVersion, &d.SSLEnabled, &d.Active, &d.CreatedAt)
	return &d, err
}

func (db *DB) GetAllDomains() ([]Domain, error) {
	rows, err := db.Query(
		"SELECT id, name, document_root, php_version, ssl_enabled, active, created_at FROM domains ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		rows.Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.PHPVersion, &d.SSLEnabled, &d.Active, &d.CreatedAt)
		domains = append(domains, d)
	}
	return domains, nil
}

func (db *DB) UpdateDomain(id int64, phpVersion string, sslEnabled, active bool) error {
	_, err := db.Exec(
		"UPDATE domains SET php_version = ?, ssl_enabled = ?, active = ? WHERE id = ?",
		phpVersion, sslEnabled, active, id,
	)
	return err
}

func (db *DB) DeleteDomain(id int64) error {
	_, err := db.Exec("DELETE FROM domains WHERE id = ?", id)
	return err
}

// Database operations
func (db *DB) CreateDatabase(name, username string, domainID *int64) (*Database, error) {
	result, err := db.Exec(
		"INSERT INTO databases (name, username, domain_id) VALUES (?, ?, ?)",
		name, username, domainID,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetDatabase(id)
}

func (db *DB) GetDatabase(id int64) (*Database, error) {
	var d Database
	err := db.QueryRow(
		"SELECT id, name, username, domain_id, created_at FROM databases WHERE id = ?", id,
	).Scan(&d.ID, &d.Name, &d.Username, &d.DomainID, &d.CreatedAt)
	return &d, err
}

func (db *DB) GetAllDatabases() ([]Database, error) {
	rows, err := db.Query("SELECT id, name, username, domain_id, created_at FROM databases ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []Database
	for rows.Next() {
		var d Database
		rows.Scan(&d.ID, &d.Name, &d.Username, &d.DomainID, &d.CreatedAt)
		databases = append(databases, d)
	}
	return databases, nil
}

func (db *DB) DeleteDatabase(id int64) error {
	_, err := db.Exec("DELETE FROM databases WHERE id = ?", id)
	return err
}

// WordPress operations
func (db *DB) CreateWordPress(domainID int64, title, adminUser, dbName, version string) (*WordPress, error) {
	result, err := db.Exec(
		"INSERT INTO wordpress (domain_id, title, admin_user, db_name, version) VALUES (?, ?, ?, ?, ?)",
		domainID, title, adminUser, dbName, version,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetWordPress(id)
}

func (db *DB) GetWordPress(id int64) (*WordPress, error) {
	var wp WordPress
	err := db.QueryRow(
		"SELECT id, domain_id, title, admin_user, db_name, version, created_at FROM wordpress WHERE id = ?", id,
	).Scan(&wp.ID, &wp.DomainID, &wp.Title, &wp.AdminUser, &wp.DBName, &wp.Version, &wp.CreatedAt)
	return &wp, err
}

func (db *DB) GetWordPressByDomain(domainID int64) (*WordPress, error) {
	var wp WordPress
	err := db.QueryRow(
		"SELECT id, domain_id, title, admin_user, db_name, version, created_at FROM wordpress WHERE domain_id = ?", domainID,
	).Scan(&wp.ID, &wp.DomainID, &wp.Title, &wp.AdminUser, &wp.DBName, &wp.Version, &wp.CreatedAt)
	return &wp, err
}

func (db *DB) GetAllWordPress() ([]WordPress, error) {
	rows, err := db.Query("SELECT id, domain_id, title, admin_user, db_name, version, created_at FROM wordpress")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []WordPress
	for rows.Next() {
		var wp WordPress
		rows.Scan(&wp.ID, &wp.DomainID, &wp.Title, &wp.AdminUser, &wp.DBName, &wp.Version, &wp.CreatedAt)
		sites = append(sites, wp)
	}
	return sites, nil
}

func (db *DB) DeleteWordPress(id int64) error {
	_, err := db.Exec("DELETE FROM wordpress WHERE id = ?", id)
	return err
}

// Backup operations
func (db *DB) CreateBackup(name, backupType string, domainID *int64, filePath string, size int64) (*Backup, error) {
	result, err := db.Exec(
		"INSERT INTO backups (name, type, domain_id, file_path, size) VALUES (?, ?, ?, ?, ?)",
		name, backupType, domainID, filePath, size,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetBackup(id)
}

func (db *DB) GetBackup(id int64) (*Backup, error) {
	var b Backup
	err := db.QueryRow(
		"SELECT id, name, type, domain_id, file_path, size, status, created_at FROM backups WHERE id = ?", id,
	).Scan(&b.ID, &b.Name, &b.Type, &b.DomainID, &b.FilePath, &b.Size, &b.Status, &b.CreatedAt)
	return &b, err
}

func (db *DB) GetAllBackups() ([]Backup, error) {
	rows, err := db.Query(
		"SELECT id, name, type, domain_id, file_path, size, status, created_at FROM backups ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var b Backup
		rows.Scan(&b.ID, &b.Name, &b.Type, &b.DomainID, &b.FilePath, &b.Size, &b.Status, &b.CreatedAt)
		backups = append(backups, b)
	}
	return backups, nil
}

func (db *DB) DeleteBackup(id int64) error {
	_, err := db.Exec("DELETE FROM backups WHERE id = ?", id)
	return err
}

// Stats
func (db *DB) GetStats() map[string]int {
	stats := make(map[string]int)

	db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&stats["domains"])
	db.QueryRow("SELECT COUNT(*) FROM databases").Scan(&stats["databases"])
	db.QueryRow("SELECT COUNT(*) FROM wordpress").Scan(&stats["wordpress"])
	db.QueryRow("SELECT COUNT(*) FROM backups").Scan(&stats["backups"])

	return stats
}
