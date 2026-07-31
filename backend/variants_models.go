package main

type VariantOption struct {
	ID        string `json:"id"`
	Value     string `json:"value"`
	SortOrder int    `json:"sort_order"`
}

type VariantType struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Position int             `json:"position"`
	Options  []VariantOption `json:"options"`
}

type VariantSku struct {
	ID        string  `json:"id"`
	Option1ID *string `json:"option1_id,omitempty"`
	Option2ID *string `json:"option2_id,omitempty"`
	Label     string  `json:"label"`
	Price     *string `json:"price,omitempty"`
	Quantity  int     `json:"quantity"`
}

type VariantTypeRequest struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

type VariantSkuRequest struct {
	OptionValues []string `json:"option_values"`
	Price        *string  `json:"price"`
	Quantity     int      `json:"quantity"`
}

type ReplaceVariantsRequest struct {
	Types []VariantTypeRequest `json:"types"`
	Skus  []VariantSkuRequest  `json:"skus"`
}
