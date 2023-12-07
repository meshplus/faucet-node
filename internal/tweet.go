package internal

import (
	"encoding/json"
	"faucet/global"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type APIResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func (c *Client) TweetReqCheck(tweetURL string, addr string) (int, string) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// 发起HTTP GET请求，替换为你的实际URL
	queryParams := url.Values{}
	queryParams.Add("tweetUrl", tweetURL)
	queryParams.Add("addr", addr)

	url := c.Config.Scrapper.ScrapperAddr
	fullURL := fmt.Sprintf("%s?%s", url, queryParams.Encode())
	resp, err := client.Get(fullURL)

	if err != nil {
		fmt.Println("HTTP请求发生错误:", err)
		return global.ScrapperErrCode, global.ScrapperErrMsg
	}
	defer resp.Body.Close()
	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return global.ScrapperErrCode, global.ScrapperErrMsg
	}

	// 根据HTTP状态码判断是否成功
	if resp.StatusCode == http.StatusOK {
		// 处理成功的情况
		fmt.Println("成功收到响应:", string(body))
		var apiResp APIResponse

		// 解析JSON数据到结构体
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			fmt.Println("解析JSON发生错误:", err)
			return global.ScrapperErrCode, global.ScrapperErrMsg
		}
		// 打印结果
		fmt.Printf("消息: %s\n", apiResp.Message)
		fmt.Printf("成功标志: %v\n", apiResp.Success)
		if apiResp.Success {
			return global.SUCCESS, apiResp.Message
		} else {
			switch apiResp.Message {
			case "The address is not in the tweet":
				return global.TweetAddrErrCode, global.TweetAddrErrMsg
			case "Err quote tweet", "No tweet content":
				return global.TweetLinkErrCode, global.TweetLinkErrMsg
			case "Err quote tweet time", "Expired tweet":
				return global.TweetTimeErrCode, global.TweetTimeErrMsg
			default:
				return global.ScrapperErrCode, global.ScrapperErrMsg
			}
		}
	} else {
		return global.ScrapperErrCode, global.ScrapperErrMsg
	}

}
