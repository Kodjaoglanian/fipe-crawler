package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"fipecrawler/internal/crawler"
	"fipecrawler/internal/repository"
)

type Handler struct {
	Crawler    *crawler.Client
	Repository *repository.Repository
}

func New(c *crawler.Client, r *repository.Repository) *Handler {
	return &Handler{Crawler: c, Repository: r}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Index)
	r.GET("/tabelas", h.GetTabelas)
	r.GET("/marcas", h.GetMarcas)
	r.GET("/modelos", h.GetModelos)
	r.POST("/extrair/marcas", h.ExtractMarcas)
	r.POST("/extrair/modelos", h.ExtractModelos)
	r.POST("/extrair/veiculos", h.ExtractVeiculos)
	r.POST("/extrair/tudo", h.ExtractAll)
	r.GET("/veiculos", h.GetVeiculos)
	r.GET("/veiculos/csv", h.GetVeiculosCSV)
	r.GET("/veiculos/search", h.SearchVeiculos)
	r.GET("/tabelas/salvas", h.GetTabelasSalvas)
}

func (h *Handler) Index(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"msg": "FIPE Crawler API"})
}

func (h *Handler) GetTabelas(c *gin.Context) {
	tables, err := h.Crawler.GetTabelas()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tables)
}

func (h *Handler) GetMarcas(c *gin.Context) {
	tabelaID, _ := strconv.Atoi(c.Query("tabela_id"))
	tipo, _ := strconv.Atoi(c.Query("tipo"))
	if tabelaID == 0 || tipo == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tabela_id and tipo required"})
		return
	}
	brands, err := h.Crawler.GetMarcas(tabelaID, tipo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, brands)
}

func (h *Handler) GetModelos(c *gin.Context) {
	tabelaID, _ := strconv.Atoi(c.Query("tabela_id"))
	tipo, _ := strconv.Atoi(c.Query("tipo"))
	marcaID, _ := strconv.Atoi(c.Query("marca_id"))
	if tabelaID == 0 || tipo == 0 || marcaID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tabela_id, tipo and marca_id required"})
		return
	}
	models, err := h.Crawler.GetModelos(tabelaID, tipo, marcaID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

func (h *Handler) ExtractMarcas(c *gin.Context) {
	var req struct {
		TabelaID int `json:"tabela_id"`
		Tipo     int `json:"tipo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	brands, err := h.Crawler.GetMarcas(req.TabelaID, req.Tipo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, brands)
}

func (h *Handler) ExtractModelos(c *gin.Context) {
	var req struct {
		TabelaID int `json:"tabela_id"`
		Tipo     int `json:"tipo"`
		MarcaID  int `json:"marca_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	models, err := h.Crawler.GetModelos(req.TabelaID, req.Tipo, req.MarcaID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

func (h *Handler) ExtractVeiculos(c *gin.Context) {
	var req struct {
		TabelaID int `json:"tabela_id"`
		Tipo     int `json:"tipo"`
		MarcaID  int `json:"marca_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	vehicles, err := h.Crawler.ExtractAllForMarca(req.TabelaID, req.Tipo, req.MarcaID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.Repository.SaveVeiculos(c.Request.Context(), vehicles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"saved": len(vehicles)})
}

func (h *Handler) ExtractAll(c *gin.Context) {
	var req struct {
		Ano  string `json:"ano"`
		Mes  string `json:"mes"`
		Tipo int    `json:"tipo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mes := fmt.Sprintf("%02d", parseInt(req.Mes))
	table, err := h.Crawler.GetTabelaByAnoMes(req.Ano, mes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	vehicles, err := h.Crawler.ExtractAll(table.ID, req.Tipo)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if err := h.Repository.SaveVeiculos(c.Request.Context(), vehicles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tabela_id": table.ID, "periodo": fmt.Sprintf("%s/%s", mes, req.Ano), "tipo": req.Tipo, "total": len(vehicles)})
}

func (h *Handler) GetVeiculos(c *gin.Context) {
	tabelaID, _ := strconv.Atoi(c.Query("tabela_id"))
	tipo, _ := strconv.Atoi(c.Query("tipo"))
	if tabelaID == 0 || tipo == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tabela_id and tipo required"})
		return
	}
	vehicles, err := h.Repository.FindVeiculosByTabelaAndTipo(c.Request.Context(), tabelaID, tipo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vehicles)
}

func (h *Handler) GetVeiculosCSV(c *gin.Context) {
	tabelaID, _ := strconv.Atoi(c.Query("tabela_id"))
	tipo, _ := strconv.Atoi(c.Query("tipo"))
	if tabelaID == 0 || tipo == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tabela_id and tipo required"})
		return
	}
	vehicles, err := h.Repository.FindVeiculosByTabelaAndTipo(c.Request.Context(), tabelaID, tipo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(repository.CSVHeader())
	for _, v := range vehicles {
		_ = w.Write(repository.VehicleToCSV(v))
	}
	w.Flush()
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=fipe_%d_%d.csv", tabelaID, tipo))
	c.String(http.StatusOK, buf.String())
}

func (h *Handler) SearchVeiculos(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q required"})
		return
	}
	vehicles, err := h.Repository.SearchVeiculos(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vehicles)
}

func (h *Handler) GetTabelasSalvas(c *gin.Context) {
	tables, err := h.Repository.FindTabelas(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": tables})
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
