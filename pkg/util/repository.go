package util

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type UtilRepositoryAdapter interface {
	GetApiAtributes()
	AddAttribute(key, name string) error
	RetrieveApiFields(apiType string) ([]byte, error)
	ExecQuery(query string) error
}
type UtilRepository struct {
	sql *sql.DB
}

func (s *UtilRepository) GetApiAttributes() {

}

type ApiKey struct {
	Key       string `json:"key"`
	Condition string `json:"condition"`
}

func (s *UtilRepository) RetrieveApiKeys(apiType string) ([]ApiKey, error) {
	query := `SELECT keys FROM api_attributes WHERE api = $1 AND type ='api'`

	var raw []byte
	if err := s.sql.QueryRow(query, apiType).Scan(&raw); err != nil {
		return nil, err
	}

	var keys []ApiKey
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}
func (s *UtilRepository) RetrieveApiFields(apiType string) ([]byte, error) {
	query := `SELECT fields FROM api_attributes WHERE api = $1 AND type = 'update'`

	var raw []byte
	if err := s.sql.QueryRow(query, apiType).Scan(&raw); err != nil {
		return nil, err
	}

	return raw, nil
}

func (s *UtilRepository) AddAttribute(key, name string) error {
	query := `INSERT INTO attributes (type, name) VALUES ($1, $2)`
	fmt.Printf("Executing query: INSERT INTO attributes (type, name) VALUES ('%s', '%s')\n", key, name)
	_, err := s.sql.Exec(query, key, name)
	fmt.Println("sql insert errr---", err)
	return err
}
func (s *UtilRepository) ExecQuery(query string) error {
	fmt.Printf("Executing query: %s\n", fmt.Sprintf(query))

	_, err := s.sql.Exec(query)
	if err != nil {
		fmt.Println("SQL execution error:", err)
		return err
	}
	return nil
}
