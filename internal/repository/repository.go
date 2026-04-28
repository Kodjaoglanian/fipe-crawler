package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fipecrawler/internal/models"
)

// Repository handles database operations
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveVeiculos inserts vehicles with conflict ignore
func (r *Repository) SaveVeiculos(ctx context.Context, vehicles []models.Vehicle) error {
	batch := &pgx.Batch{}
	for _, v := range vehicles {
		batch.Queue(`
			INSERT INTO veiculo (fipe_cod, tabela_id, anoref, mesref, tipo, marca_id, marca, modelo_id, modelo, anomod, comb_cod, comb_sigla, comb, valor)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (fipe_cod, anomod, comb_cod) DO NOTHING
		`, v.FipeCode, v.TabelaID, v.AnoRef, v.MesRef, v.Tipo, v.MarcaID, v.Marca, v.ModeloID, v.Modelo, v.AnoMod, v.CombCod, v.CombSigla, v.Comb, v.Valor)
	}
	br := r.pool.SendBatch(ctx, batch)
	return br.Close()
}

// FindVeiculosByTabelaAndTipo returns saved vehicles
func (r *Repository) FindVeiculosByTabelaAndTipo(ctx context.Context, tabelaID, tipo int) ([]models.Vehicle, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, fipe_cod, tabela_id, anoref, mesref, tipo, marca_id, marca, modelo_id, modelo, anomod, comb_cod, comb_sigla, comb, valor, created_at
		FROM veiculo WHERE tabela_id = $1 AND tipo = $2 ORDER BY marca, modelo, anomod
	`, tabelaID, tipo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Vehicle])
}

// FindVeiculos returns vehicles by year/month/type
func (r *Repository) FindVeiculos(ctx context.Context, anoRef, mesRef, tipo int) ([]models.Vehicle, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, fipe_cod, tabela_id, anoref, mesref, tipo, marca_id, marca, modelo_id, modelo, anomod, comb_cod, comb_sigla, comb, valor, created_at
		FROM veiculo WHERE anoref = $1 AND mesref = $2 AND tipo = $3 ORDER BY marca, modelo, anomod
	`, anoRef, mesRef, tipo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Vehicle])
}

// FindTabelas returns distinct tables present in the database
func (r *Repository) FindTabelas(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT tabela_id, anoref, mesref, tipo FROM veiculo ORDER BY anoref DESC, mesref DESC, tipo
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var tabelaID, anoref, mesref, tipo int
		if err := rows.Scan(&tabelaID, &anoref, &mesref, &tipo); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":  fmt.Sprintf("%d-%d", tabelaID, tipo),
			"lbl": fmt.Sprintf("%s/%d - %s", monthName(mesref), anoref, typeName(tipo)),
		})
	}
	return results, rows.Err()
}

func monthName(m int) string {
	names := map[int]string{
		1: "janeiro", 2: "fevereiro", 3: "março", 4: "abril",
		5: "maio", 6: "junho", 7: "julho", 8: "agosto",
		9: "setembro", 10: "outubro", 11: "novembro", 12: "dezembro",
	}
	return names[m]
}

func typeName(t int) string {
	names := map[int]string{1: "Carro", 2: "Moto", 3: "Caminhão"}
	return names[t]
}

// CSVHeader returns the CSV column names
func CSVHeader() []string {
	return []string{"fipe_cod", "tabela_id", "anoref", "mesref", "tipo", "marca_id", "marca", "modelo_id", "modelo", "anomod", "comb_cod", "comb_sigla", "comb", "valor"}
}

// VehicleToCSV converts a vehicle to CSV row
func VehicleToCSV(v models.Vehicle) []string {
	return []string{
		v.FipeCode, fmt.Sprintf("%d", v.TabelaID), fmt.Sprintf("%d", v.AnoRef),
		fmt.Sprintf("%d", v.MesRef), fmt.Sprintf("%d", v.Tipo), fmt.Sprintf("%d", v.MarcaID),
		v.Marca, fmt.Sprintf("%d", v.ModeloID), v.Modelo, fmt.Sprintf("%d", v.AnoMod),
		fmt.Sprintf("%d", v.CombCod), v.CombSigla, v.Comb, fmt.Sprintf("%d", v.Valor),
	}
}

// RunMigrations executes schema migrations
func (r *Repository) RunMigrations(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS veiculo (
			id SERIAL PRIMARY KEY,
			fipe_cod VARCHAR(10),
			tabela_id INTEGER NOT NULL,
			anoref SMALLINT NOT NULL,
			mesref SMALLINT NOT NULL,
			tipo SMALLINT NOT NULL,
			marca_id INTEGER NOT NULL,
			marca VARCHAR(50),
			modelo_id INTEGER NOT NULL,
			modelo VARCHAR(50) NOT NULL,
			anomod SMALLINT NOT NULL,
			comb_cod SMALLINT NOT NULL,
			comb_sigla CHAR(1) NOT NULL,
			comb VARCHAR(10) NOT NULL,
			valor INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_veiculo_unique ON veiculo (fipe_cod, anomod, comb_cod);
	`)
	return err
}

// SearchVeiculos searches vehicles by term
func (r *Repository) SearchVeiculos(ctx context.Context, query string) ([]models.Vehicle, error) {
	q := "%" + strings.ToLower(query) + "%"
	rows, err := r.pool.Query(ctx, `
		SELECT id, fipe_cod, tabela_id, anoref, mesref, tipo, marca_id, marca, modelo_id, modelo, anomod, comb_cod, comb_sigla, comb, valor, created_at
		FROM veiculo WHERE LOWER(marca) LIKE $1 OR LOWER(modelo) LIKE $1 OR LOWER(fipe_cod) LIKE $1
		ORDER BY marca, modelo, anomod
		LIMIT 100
	`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Vehicle])
}
