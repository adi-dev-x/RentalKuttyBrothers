package util

import (
	"fmt"
	"github.com/labstack/echo/v4"
)

// Define a named KeyValue type
type KeyValue struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Condition string `json:"condition"`
}

type Handler struct {
	KeyValue []KeyValue `json:"key_value"`
	Limit    string     `json:"limit"`
	Offset   string     `json:"offset"`
	Table    string     `json:"table"`
	Joins    string     `json:"joins"`
}
type ApiFormaters struct {
	Repo UtilRepository
}

func (h *ApiFormaters) Initiator(c echo.Context, apiType string) (string, error) {
	keys, err := h.Repo.RetrieveApiKeys(apiType)
	fmt.Println("keysss----init", keys)
	if err != nil {
		fmt.Println("this-- err", err.Error())
		return "", err
	}

	handler := Handler{}
	handler.KeyValue = h.extractParams(c, keys, true)
	fmt.Println("this ---- handler end-", handler.KeyValue)
	limit, page := h.extractLimiters(c)
	handler.Limit = limit
	handler.Offset = page
	end, _ := QueryBuilder(handler, "")

	return end, nil
}
func (h *ApiFormaters) extractLimiters(c echo.Context) (limit, page string) {
	limit = c.QueryParam("limit")
	page = c.QueryParam("page")
	return
}

func (h *ApiFormaters) extractParams(c echo.Context, keys []ApiKey, fromQuery bool) []KeyValue {
	var result []KeyValue
	fmt.Println("----keysssss", keys)
	for _, k := range keys {
		var v string
		if fromQuery {

			v = c.QueryParam(k.Key)
			fmt.Println("query and valll---", k.Key, "--", v)
		}
		//} else {
		//	v = c.Param(k.Key)
		//}

		// only append if value exists
		if v != "" {
			result = append(result, KeyValue{
				Key:       k.Key,
				Value:     v,
				Condition: k.Condition,
			})
		}
	}
	fmt.Println("thisss -is --ress--", result)
	return result
}
