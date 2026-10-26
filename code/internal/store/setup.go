package store

import (
	"regexp"
	"strings"
)

type Class struct {
	ID   int64
	Name string
}

type Subject struct {
	ID   int64
	Name string
}

// Term is one semester/term of an academic year.
type Term struct {
	ID   int64
	Year string
	Name string
}

func (t Term) Label() string { return t.Year + " - " + t.Name }

func (s *Store) listNamed(table string) ([]Class, error) {
	rows, err := s.db.Query(`SELECT id, name FROM ` + table + ` ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Class
	for rows.Next() {
		var c Class
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) createNamed(table, field, name string) (int64, error) {
	name, err := cleanText(name, field, 60)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO `+table+` (name) VALUES (?)`, name)
	if isUniqueViolation(err) {
		return 0, invalid("%s %q already exists.", field, name)
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListClasses() ([]Class, error) { return s.listNamed("classes") }

func (s *Store) CreateClass(name string) (int64, error) {
	return s.createNamed("classes", "Class", name)
}

func (s *Store) ListSubjects() ([]Subject, error) {
	cs, err := s.listNamed("subjects")
	out := make([]Subject, len(cs))
	for i, c := range cs {
		out[i] = Subject(c)
	}
	return out, err
}

func (s *Store) CreateSubject(name string) (int64, error) {
	return s.createNamed("subjects", "Subject", name)
}

var yearRe = regexp.MustCompile(`^\d{4}(/\d{4})?$`)

// CreateTerm adds a term, creating its academic year if needed.
func (s *Store) CreateTerm(year, name string) (int64, error) {
	year = strings.TrimSpace(year)
	if !yearRe.MatchString(year) {
		return 0, invalid("Academic year must look like 2026 or 2025/2026.")
	}
	name, err := cleanText(name, "Term name", 40)
	if err != nil {
		return 0, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO academic_years (name) VALUES (?) ON CONFLICT (name) DO NOTHING`, year); err != nil {
		return 0, err
	}
	res, err := tx.Exec(`INSERT INTO terms (academic_year_id, name)
		SELECT id, ? FROM academic_years WHERE name = ?`, name, year)
	if isUniqueViolation(err) {
		return 0, invalid("%s %s already exists.", year, name)
	}
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

const termSelect = `SELECT t.id, y.name, t.name FROM terms t JOIN academic_years y ON y.id = t.academic_year_id`

// ListTerms returns terms newest first.
func (s *Store) ListTerms() ([]Term, error) {
	rows, err := s.db.Query(termSelect + ` ORDER BY y.name DESC, t.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Term
	for rows.Next() {
		var t Term
		if err := rows.Scan(&t.ID, &t.Year, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) TermByID(id int64) (*Term, error) {
	var t Term
	err := s.db.QueryRow(termSelect+` WHERE t.id = ?`, id).Scan(&t.ID, &t.Year, &t.Name)
	if err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}
