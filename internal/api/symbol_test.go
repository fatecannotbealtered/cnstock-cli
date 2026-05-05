package api

import "testing"

func TestDetectMarket(t *testing.T) {
	tests := []struct {
		symbol string
		want   string
	}{
		{"sh600519", MarketCN},
		{"sz000858", MarketCN},
		{"bj430047", MarketCN},
		{"hk00700", MarketHK},
		{"usAAPL", MarketUS},
		{"USMSFT", MarketUS},
		{"unknown", MarketUnknown},
	}
	for _, tt := range tests {
		if got := DetectMarket(tt.symbol); got != tt.want {
			t.Errorf("DetectMarket(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"600519", "sh600519"}, // 沪市主板 6 开头
		{"000858", "sz000858"}, // 深市主板 0 开头
		{"300001", "sz300001"}, // 创业板 3 开头
		{"430047", "bj430047"}, // 北交所 4 开头
		{"830000", "bj830000"}, // 北交所 8 开头
		{"510050", "sh510050"}, // 沪市 ETF 5 开头
		{"900901", "sh900901"}, // 沪市 B 股 9 开头
		{"123001", "sz123001"}, // 深市可转债 1 开头
		{"200001", "sz200001"}, // 深市 B 股 2 开头
		{"sh600519", "sh600519"},
		{"SZ000858", "sz000858"},
		{"00700", "hk00700"},
		{"700", "hk00700"},
		{"hk00700", "hk00700"},
		{"AAPL", "usAAPL"},
		{"usAAPL", "usAAPL"},
		{"USMSFT", "usMSFT"},
		{"TSLA", "usTSLA"},
		{"BRK.B", "usBRK.B"}, // 美股带点代码
	}
	for _, tt := range tests {
		got, err := NormalizeSymbol(tt.input)
		if err != nil {
			t.Errorf("NormalizeSymbol(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeSymbol(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeSymbolEmpty(t *testing.T) {
	_, err := NormalizeSymbol("")
	if err == nil {
		t.Error("NormalizeSymbol(\"\") should return error")
	}
}

func TestNormalizeSymbols(t *testing.T) {
	got, err := NormalizeSymbols("600519,hk00700,usAAPL")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sh600519", "hk00700", "usAAPL"}
	if len(got) != len(want) {
		t.Fatalf("got %d symbols, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeAdj(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"qfq", "qfq"},
		{"hfq", "hfq"},
		{"none", ""},
		{"", ""},
		{"raw", ""},
	}
	for _, tt := range tests {
		got, err := NormalizeAdj(tt.input)
		if err != nil {
			t.Errorf("NormalizeAdj(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeAdj(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
