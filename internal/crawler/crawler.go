package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fipecrawler/internal/models"
)

var urls = map[string]string{
	"tabelas":    "https://veiculos.fipe.org.br/api/veiculos/ConsultarTabelaDeReferencia",
	"marcas":     "https://veiculos.fipe.org.br/api/veiculos/ConsultarMarcas",
	"modelos":    "https://veiculos.fipe.org.br/api/veiculos/ConsultarModelos",
	"anoModelos": "https://veiculos.fipe.org.br/api/veiculos/ConsultarAnoModelo",
	"veiculo":    "https://veiculos.fipe.org.br/api/veiculos/ConsultarValorComTodosParametros",
}

var tipos = map[int]string{1: "carro", 2: "moto", 3: "caminhao"}
var tiposFull = map[int]string{1: "Carro", 2: "Moto", 3: "Caminhão"}

var meses = map[string]string{
	"janeiro": "01", "fevereiro": "02", "março": "03", "abril": "04",
	"maio": "05", "junho": "06", "julho": "07", "agosto": "08",
	"setembro": "09", "outubro": "10", "novembro": "11", "dezembro": "12",
}

var combustiveis = map[int]string{1: "Gasolina", 2: "Álcool", 3: "Diesel", 4: "Flex"}

type Client struct{ httpClient *http.Client }

func NewClient() *Client { return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}} }

func GetTipos() map[int]string        { return tiposFull }
func GetCombustiveis() map[int]string { return combustiveis }

func (c *Client) httpPost(endpoint string, params map[string]interface{}) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, fmt.Sprintf("%v", v))
	}
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.8,en-US;q=0.6,en;q=0.4")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Host", "veiculos.fipe.org.br")
	req.Header.Set("Origin", "https://veiculos.fipe.org.br")
	req.Header.Set("Referer", "https://veiculos.fipe.org.br/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) GetTabelas() ([]models.Table, error) {
	data, err := c.httpPost(urls["tabelas"], map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Codigo int    `json:"Codigo"`
		Mes    string `json:"Mes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var tables []models.Table
	for _, r := range raw {
		parts := strings.Split(r.Mes, "/")
		if len(parts) != 2 {
			continue
		}
		tables = append(tables, models.Table{
			ID: r.Codigo, Lbl: r.Mes, Ano: strings.TrimSpace(parts[1]),
			Mes: meses[strings.ToLower(strings.TrimSpace(parts[0]))],
		})
	}
	return tables, nil
}

func (c *Client) GetTabelaByAnoMes(ano, mes string) (*models.Table, error) {
	tables, err := c.GetTabelas()
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		if t.Ano == ano && t.Mes == mes {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("tabela nao encontrada para %s/%s", mes, ano)
}

func (c *Client) GetMarcas(tabelaID, tipo int) ([]models.Brand, error) {
	data, err := c.httpPost(urls["marcas"], map[string]interface{}{
		"codigoTabelaReferencia": tabelaID, "codigoTipoVeiculo": tipo,
	})
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Value json.Number `json:"Value"`
		Label string      `json:"Label"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var brands []models.Brand
	for _, r := range raw {
		id, _ := strconv.Atoi(r.Value.String())
		brands = append(brands, models.Brand{ID: id, Label: r.Label, Tipo: tipo})
	}
	return brands, nil
}

func (c *Client) GetModelos(tabelaID, tipo, marcaID int) ([]models.Model, error) {
	data, err := c.httpPost(urls["modelos"], map[string]interface{}{
		"codigoTipoVeiculo": tipo, "codigoTabelaReferencia": tabelaID,
		"codigoModelo": "", "codigoMarca": marcaID, "ano": "",
		"codigoTipoCombustivel": "", "anoModelo": "", "modeloCodigoExterno": "",
	})
	if err != nil {
		return nil, err
	}
	var result struct {
		Modelos []struct {
			Value json.Number `json:"Value"`
			Label string      `json:"Label"`
		} `json:"Modelos"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	var list []models.Model
	for _, r := range result.Modelos {
		id, _ := strconv.Atoi(r.Value.String())
		list = append(list, models.Model{ID: id, Label: r.Label, Tipo: tipo})
	}
	return list, nil
}

func (c *Client) GetAnoModelos(tabelaID, tipo, marcaID, modeloID int) ([]models.YearModel, error) {
	data, err := c.httpPost(urls["anoModelos"], map[string]interface{}{
		"codigoTipoVeiculo": tipo, "codigoTabelaReferencia": tabelaID,
		"codigoModelo": modeloID, "codigoMarca": marcaID, "ano": "",
		"codigoTipoCombustivel": "", "anoModelo": "", "modeloCodigoExterno": "",
	})
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Value string `json:"Value"`
		Label string `json:"Label"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var years []models.YearModel
	for _, r := range raw {
		parts := strings.Split(r.Value, "-")
		if len(parts) != 2 {
			continue
		}
		years = append(years, models.YearModel{ID: r.Value, Label: r.Label, Ano: parts[0], Comb: parts[1]})
	}
	return years, nil
}

func (c *Client) GetVeiculo(tabelaID, tipo, marcaID, modeloID int, combustivel, ano string) (*models.Vehicle, error) {
	params := map[string]interface{}{
		"codigoTipoVeiculo": tipo, "codigoTabelaReferencia": tabelaID,
		"codigoModelo": modeloID, "codigoMarca": marcaID,
		"codigoTipoCombustivel": combustivel, "anoModelo": ano,
		"modeloCodigoExterno": "", "tipoVeiculo": tipos[tipo], "tipoConsulta": "tradicional",
	}
	data, err := c.httpPost(urls["veiculo"], params)
	if err != nil {
		return nil, err
	}
	var resp models.FipeVehicleResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Valor == "" {
		return nil, nil
	}
	valor := parseValor(resp.Valor)
	anoMod := resp.AnoModelo
	combCod, _ := strconv.Atoi(combustivel)
	mesRef, anoRef := "", ""
	if resp.MesReferencia != "" {
		parts := strings.Split(resp.MesReferencia, " ")
		if len(parts) >= 3 {
			mesRef = meses[strings.ToLower(parts[0])]
			anoRef = strings.TrimSpace(parts[2])
		}
	}
	return &models.Vehicle{
		TabelaID: tabelaID, AnoRef: parseInt(anoRef), MesRef: parseInt(mesRef), Tipo: tipo,
		FipeCode: strings.TrimSpace(resp.CodigoFipe), MarcaID: marcaID,
		Marca: strings.TrimSpace(resp.Marca), ModeloID: modeloID,
		Modelo: strings.TrimSpace(resp.Modelo), AnoMod: anoMod,
		CombCod: combCod, CombSigla: strings.TrimSpace(resp.SiglaCombustivel),
		Comb: combustiveis[combCod], Valor: valor,
	}, nil
}

func (c *Client) ExtractAllForMarca(tabelaID, tipo, marcaID int) ([]models.Vehicle, error) {
	var vehicles []models.Vehicle
	modelos, err := c.GetModelos(tabelaID, tipo, marcaID)
	if err != nil {
		return nil, err
	}
	for _, modelo := range modelos {
		anos, err := c.GetAnoModelos(tabelaID, tipo, marcaID, modelo.ID)
		if err != nil {
			continue
		}
		for _, ano := range anos {
			v, err := c.GetVeiculo(tabelaID, tipo, marcaID, modelo.ID, ano.Comb, ano.Ano)
			if err != nil || v == nil {
				continue
			}
			vehicles = append(vehicles, *v)
		}
	}
	return vehicles, nil
}

func (c *Client) ExtractAll(tabelaID, tipo int) ([]models.Vehicle, error) {
	var vehicles []models.Vehicle
	marcas, err := c.GetMarcas(tabelaID, tipo)
	if err != nil {
		return nil, err
	}
	for _, marca := range marcas {
		vs, err := c.ExtractAllForMarca(tabelaID, tipo, marca.ID)
		if err != nil {
			continue
		}
		vehicles = append(vehicles, vs...)
	}
	return vehicles, nil
}

func parseValor(v string) int {
	v = strings.ReplaceAll(v, "R$ ", "")
	v = strings.ReplaceAll(v, ".", "")
	v = strings.ReplaceAll(v, ",", ".")
	f, _ := strconv.ParseFloat(v, 64)
	return int(f)
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
