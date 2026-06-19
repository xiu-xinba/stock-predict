package fund

// Repository 定义了基金领域的基础数据访问接口。
type Repository interface {
	// ListFunds 返回所有已加载的基金列表。
	ListFunds() []FundItem
	// FindFund 根据基金代码查找基金，第二个返回值表示是否找到。
	FindFund(code string) (FundItem, bool)
	// CountFunds 返回已加载的基金总数。
	CountFunds() int
}

// CoverageRepository 扩展 Repository，增加数据覆盖情况查询能力。
type CoverageRepository interface {
	Repository
	// CoverageReport 返回基金数据的覆盖情况报告。
	CoverageReport() *CoverageReport
}

// PersistenceRepository 扩展 CoverageRepository，增加持久化读写能力。
type PersistenceRepository interface {
	CoverageRepository
	// LoadFunds 从持久化存储加载全部基金数据。
	LoadFunds() []FundItem
	// SaveFunds 将基金数据批量写入持久化存储。
	SaveFunds(funds []FundItem) error
	// GetFunds 根据基金代码列表批量查询基金数据。
	GetFunds(codes []string) []FundItem
	// AddFund 添加单只基金到持久化存储。
	AddFund(fund FundItem) error
	// RemoveFund 根据基金代码从持久化存储中删除基金。
	RemoveFund(code string) error
}
