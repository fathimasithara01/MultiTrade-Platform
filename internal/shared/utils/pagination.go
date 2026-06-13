package utils

import (
	"math"

	"github.com/fathimasithara01/multitrade-platform/internal/shared/constants"
	"github.com/fathimasithara01/multitrade-platform/internal/shared/response"
)

type PaginationRequest struct {
	Page     int `json:"page" binding:"min=1"`
	PageSize int `json:"pageSize" binding:"min=1,max=100"`
}

func (p *PaginationRequest) SetDefaults() {
	if p.Page == 0 {
		p.Page = constants.DefaultPage
	}
	if p.PageSize == 0 {
		p.PageSize = constants.DefaultPageSize
	}
	if p.PageSize > constants.MaxPageSize {
		p.PageSize = constants.MaxPageSize
	}
}

func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *PaginationRequest) GetLimit() int {
	return p.PageSize
}

func CalculatePaginationMeta(page, pageSize int, totalCount int64) response.PaginationMeta {
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	return response.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
