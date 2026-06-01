package response

type Envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func Success[T any](data T) Envelope[T] {
	return Envelope[T]{Code: 0, Message: "success", Data: data}
}
