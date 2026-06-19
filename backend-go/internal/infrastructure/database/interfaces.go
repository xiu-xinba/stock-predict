package database

import (
	funddomain "stock-predict-go/internal/domain/fund"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// FundRepository 基金持久化仓库接口，委托给 domain 层的 PersistenceRepository
type FundRepository = funddomain.PersistenceRepository

// StockRepository 股票持久化仓库接口，委托给 domain 层的 Repository
type StockRepository = stockdomain.Repository
