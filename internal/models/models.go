package models

import "time"

// Vehicle represents a FIPE vehicle record
type Vehicle struct {
	ID         int       `db:"id" json:"id"`
	FipeCode   string    `db:"fipe_cod" json:"fipe_cod"`
	TabelaID   int       `db:"tabela_id" json:"tabela_id"`
	AnoRef     int       `db:"anoref" json:"anoref"`
	MesRef     int       `db:"mesref" json:"mesref"`
	Tipo       int       `db:"tipo" json:"tipo"`
	MarcaID    int       `db:"marca_id" json:"marca_id"`
	Marca      string    `db:"marca" json:"marca"`
	ModeloID   int       `db:"modelo_id" json:"modelo_id"`
	Modelo     string    `db:"modelo" json:"modelo"`
	AnoMod     int       `db:"anomod" json:"anomod"`
	CombCod    int       `db:"comb_cod" json:"comb_cod"`
	CombSigla  string    `db:"comb_sigla" json:"comb_sigla"`
	Comb       string    `db:"comb" json:"comb"`
	Valor      int       `db:"valor" json:"valor"`
	CreatedAt  time.Time `db:"created_at" json:"created_at,omitempty"`
}

// Brand represents a vehicle brand
type Brand struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Tipo  int    `json:"tipo"`
}

// Model represents a vehicle model
type Model struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Tipo  int    `json:"tipo"`
}

// YearModel represents a model year with fuel type
type YearModel struct {
	ID   string `json:"id"`
	Label string `json:"label"`
	Comb string `json:"comb"`
	Ano  string `json:"ano"`
}

// Table represents a FIPE reference table
type Table struct {
	ID  int    `json:"id"`
	Lbl string `json:"lbl"`
	Ano string `json:"ano"`
	Mes string `json:"mes"`
}

// FipeVehicleResponse represents the FIPE API vehicle response
type FipeVehicleResponse struct {
	Valor          string `json:"Valor"`
	Marca          string `json:"Marca"`
	Modelo         string `json:"Modelo"`
	AnoModelo      int    `json:"AnoModelo"`
	Combustivel    string `json:"Combustivel"`
	CodigoFipe     string `json:"CodigoFipe"`
	MesReferencia  string `json:"MesReferencia"`
	SiglaCombustivel string `json:"SiglaCombustivel"`
	TipoVeiculo    int    `json:"TipoVeiculo"`
}
