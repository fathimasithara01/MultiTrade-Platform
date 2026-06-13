package dto

type CreateAssetInput struct {
	Symbol      string  `json:"symbol"      binding:"required,max=50"`
	Name        string  `json:"name"        binding:"required,max=255"`
	Description *string `json:"description"`
	Price       string  `json:"price"       binding:"required"`
	Quantity    string  `json:"quantity"    binding:"required"`
}

type UpdateAssetInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *string `json:"price"`
	Quantity    *string `json:"quantity"`
}
