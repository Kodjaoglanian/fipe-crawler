package crawler

import (
	"testing"
)

func TestParseValor(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"R$ 50.000,00", 50000},
		{"R$ 1.234.567,89", 1234567},
		{"R$ 99,99", 99},
		{"", 0},
	}
	for _, tt := range tests {
		result := parseValor(tt.input)
		if result != tt.expected {
			t.Errorf("parseValor(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestGetTipos(t *testing.T) {
	tipos := GetTipos()
	if len(tipos) != 3 {
		t.Errorf("expected 3 tipos, got %d", len(tipos))
	}
	if tipos[1] != "Carro" {
		t.Errorf("expected tipo 1 = Carro, got %s", tipos[1])
	}
}

func TestGetCombustiveis(t *testing.T) {
	comb := GetCombustiveis()
	if len(comb) != 4 {
		t.Errorf("expected 4 combustiveis, got %d", len(comb))
	}
}
