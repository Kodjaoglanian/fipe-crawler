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

CREATE UNIQUE INDEX IF NOT EXISTS idx_veiculo_unique
    ON veiculo (fipe_cod, anomod, comb_cod);
