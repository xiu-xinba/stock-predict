namespace StockPredictApi.Configuration;

public class AppSettings
{
    public string Env { get; set; } = "development";
    public bool Debug { get; set; } = true;
    public string CorsOrigins { get; set; } = "http://localhost:5173";
    public EastMoneySettings EastMoney { get; set; } = new();
    public ModelSettings Model { get; set; } = new();
    public MarketSettings Market { get; set; } = new();
    public FundSettings Fund { get; set; } = new();
}

public class EastMoneySettings
{
    public string BaseUrl { get; set; } = "https://fund.eastmoney.com";
    public double RequestDelay { get; set; } = 0.5;
}

public class ModelSettings
{
    public string Path { get; set; } = "./models/model_v1.onnx";
    public int CacheTtl { get; set; } = 300;
}

public class MarketSettings
{
    public int CacheTtl { get; set; } = 30;
}

public class FundSettings
{
    public int CacheTtl { get; set; } = 30;
}
