import unittest
import asyncio
from unittest.mock import patch

import main
from fastapi import HTTPException


def tencent_statement(symbol, name, code, price, volume, change, change_pct, high, low):
    fields = [""] * 41
    fields[1] = name
    fields[2] = code
    fields[3] = str(price)
    fields[6] = str(volume)
    fields[30] = "20260607150000"
    fields[31] = str(change)
    fields[32] = str(change_pct)
    fields[33] = str(high)
    fields[34] = str(low)
    return f'v_{symbol}="' + "~".join(fields) + '";'


class FakeTencentResponse:
    status_code = 200

    def __init__(self, text):
        self.content = text.encode("gbk")

    def raise_for_status(self):
        return None


class AkshareServiceTest(unittest.TestCase):
    def test_service_token_is_required(self):
        previous = main.SERVICE_TOKEN
        try:
            main.SERVICE_TOKEN = "expected-token"
            with self.assertRaises(HTTPException) as context:
                main.require_service_token("Bearer wrong-token")
            self.assertEqual(context.exception.status_code, 401)
            self.assertIsNone(main.require_service_token("Bearer expected-token"))
        finally:
            main.SERVICE_TOKEN = previous

    def test_index_quote_falls_back_to_tencent_when_akshare_fails(self):
        payload = "".join(
            [
                tencent_statement("sh000001", "SSE Composite", "000001", 4027.74, 662918577, -30.04, -0.74, 4078.93, 4015.06),
                tencent_statement("sz399001", "SZ Component", "399001", 12845.2, 506790000, -11.0, -0.09, 12900.0, 12780.0),
                tencent_statement("sz399006", "ChiNext", "399006", 2634.1, 245670000, 18.2, 0.7, 2640.0, 2601.0),
            ]
        )
        response = FakeTencentResponse(payload)

        with patch("main.ak.stock_zh_index_spot_em", side_effect=RuntimeError("eastmoney unavailable")):
            with patch("main.requests.get", return_value=response):
                body = asyncio.run(main.index_quote("cn"))

        self.assertEqual(body["code"], 0)
        self.assertEqual(len(body["data"]), 3)
        self.assertEqual(body["data"][0]["code"], "000001")
        self.assertEqual(body["data"][0]["price"], 4027.74)
        self.assertEqual(body["data"][0]["change_pct"], -0.74)


if __name__ == "__main__":
    unittest.main()
