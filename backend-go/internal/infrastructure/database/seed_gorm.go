package database

import (
	funddomain "stock-predict-go/internal/domain/fund"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedFunds 将种子基金数据插入数据库。已存在的基金不会覆盖（OnConflict DoNothing）。
func SeedFunds(db *gorm.DB) error {
	items := seedFundItems()
	if len(items) == 0 {
		return nil
	}
	models := make([]Fund, 0, len(items))
	for _, f := range items {
		f.FundCode = normalizeFundCode(f.FundCode)
		if f.FundCode == "" {
			continue
		}
		models = append(models, fundDTOToModel(f))
	}
	if len(models) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models).Error
}

// seedFundItems 返回与内存版 seedFunds() 相同的种子基金数据列表。
func seedFundItems() []funddomain.FundItem {
	return []funddomain.FundItem{
		{FundCode: "000001", FundName: "华夏成长混合", FundType: "混合型", Company: "华夏基金", Manager: "阳琨", InceptionDate: "2001-12-18", RiskLevel: "中高"},
		{FundCode: "000011", FundName: "华夏大盘精选混合", FundType: "混合型", Company: "华夏基金", Manager: "陈伟彦", InceptionDate: "2004-08-11", RiskLevel: "中高"},
		{FundCode: "000021", FundName: "华夏优势增长混合", FundType: "混合型", Company: "华夏基金", Manager: "郑煜", InceptionDate: "2006-11-24", RiskLevel: "中高"},
		{FundCode: "000031", FundName: "华夏复兴混合", FundType: "混合型", Company: "华夏基金", Manager: "赵航", InceptionDate: "2007-08-10", RiskLevel: "中高"},
		{FundCode: "000041", FundName: "华夏全球精选", FundType: "QDII", Company: "华夏基金", Manager: "郑鹏", InceptionDate: "2007-10-09", RiskLevel: "高"},
		{FundCode: "000051", FundName: "华夏沪深300ETF联接", FundType: "指数型", Company: "华夏基金", Manager: "赵宇", InceptionDate: "2009-08-28", RiskLevel: "中"},
		{FundCode: "000061", FundName: "华夏盛世精选混合", FundType: "混合型", Company: "华夏基金", Manager: "张帆", InceptionDate: "2009-12-11", RiskLevel: "中高"},
		{FundCode: "000071", FundName: "华夏恒生ETF联接", FundType: "QDII", Company: "华夏基金", Manager: "张弘弢", InceptionDate: "2012-08-21", RiskLevel: "高"},
		{FundCode: "000091", FundName: "华夏移动互联混合", FundType: "混合型", Company: "华夏基金", Manager: "刘平", InceptionDate: "2013-06-27", RiskLevel: "中高"},
		{FundCode: "000101", FundName: "华夏红利混合", FundType: "混合型", Company: "华夏基金", Manager: "王怡欢", InceptionDate: "2005-06-30", RiskLevel: "中高"},
		{FundCode: "110011", FundName: "易方达中小盘混合", FundType: "混合型", Company: "易方达基金", Manager: "张坤", InceptionDate: "2008-06-19", RiskLevel: "中高"},
		{FundCode: "110022", FundName: "易方达消费行业股票", FundType: "股票型", Company: "易方达基金", Manager: "王元春", InceptionDate: "2010-08-20", RiskLevel: "高"},
		{FundCode: "005827", FundName: "易方达蓝筹精选混合", FundType: "混合型", Company: "易方达基金", Manager: "张坤", InceptionDate: "2018-09-05", RiskLevel: "中高"},
		{FundCode: "161725", FundName: "招商中证白酒指数", FundType: "指数型", Company: "招商基金", Manager: "侯昊", InceptionDate: "2015-05-27", RiskLevel: "中"},
		{FundCode: "003834", FundName: "华夏能源革新股票", FundType: "股票型", Company: "华夏基金", Manager: "郑泽鸿", InceptionDate: "2017-06-07", RiskLevel: "高"},
		{FundCode: "005911", FundName: "广发双擎升级混合", FundType: "混合型", Company: "广发基金", Manager: "刘格菘", InceptionDate: "2018-11-02", RiskLevel: "中高"},
		{FundCode: "007119", FundName: "景顺长城绩优成长混合", FundType: "混合型", Company: "景顺长城基金", Manager: "刘彦春", InceptionDate: "2019-04-11", RiskLevel: "中高"},
		{FundCode: "001938", FundName: "中欧时代先锋股票", FundType: "股票型", Company: "中欧基金", Manager: "周蔚文", InceptionDate: "2015-11-03", RiskLevel: "高"},
		{FundCode: "001156", FundName: "申万菱信新能源汽车", FundType: "混合型", Company: "申万菱信基金", Manager: "任琳娜", InceptionDate: "2015-03-12", RiskLevel: "中高"},
		{FundCode: "519736", FundName: "交银新成长混合", FundType: "混合型", Company: "交银施罗德基金", Manager: "王崇", InceptionDate: "2014-12-18", RiskLevel: "中高"},
		{FundCode: "001632", FundName: "天弘创新驱动混合", FundType: "混合型", Company: "天弘基金", Manager: "田俊维", InceptionDate: "2015-06-12", RiskLevel: "中高"},
		{FundCode: "001714", FundName: "工银前沿医疗股票", FundType: "股票型", Company: "工银瑞信基金", Manager: "赵蓓", InceptionDate: "2015-08-27", RiskLevel: "高"},
		{FundCode: "001875", FundName: "前海开源公用事业股票", FundType: "股票型", Company: "前海开源基金", Manager: "崔宸龙", InceptionDate: "2015-09-15", RiskLevel: "高"},
		{FundCode: "002190", FundName: "农银汇理新能源主题", FundType: "混合型", Company: "农银汇理基金", Manager: "邢军亮", InceptionDate: "2016-03-29", RiskLevel: "中高"},
		{FundCode: "002621", FundName: "中欧医疗创新股票", FundType: "股票型", Company: "中欧基金", Manager: "葛兰", InceptionDate: "2016-09-29", RiskLevel: "高"},
		{FundCode: "003096", FundName: "中欧医疗健康混合", FundType: "混合型", Company: "中欧基金", Manager: "葛兰", InceptionDate: "2016-09-29", RiskLevel: "中高"},
		{FundCode: "004851", FundName: "广发医疗保健股票", FundType: "股票型", Company: "广发基金", Manager: "吴兴武", InceptionDate: "2017-08-10", RiskLevel: "高"},
		{FundCode: "005818", FundName: "景顺长城新兴成长混合", FundType: "混合型", Company: "景顺长城基金", Manager: "刘彦春", InceptionDate: "2018-04-16", RiskLevel: "中高"},
		{FundCode: "006228", FundName: "中欧时代智慧混合", FundType: "混合型", Company: "中欧基金", Manager: "周蔚文", InceptionDate: "2018-10-12", RiskLevel: "中高"},
		{FundCode: "007874", FundName: "华夏科技创新混合", FundType: "混合型", Company: "华夏基金", Manager: "张帆", InceptionDate: "2019-05-06", RiskLevel: "中高"},
	}
}
