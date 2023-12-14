package global

type DirectClaimReq struct {
	Net     string `json:"net"`
	Address string `json:"address"`
}

type TweetClaimReq struct {
	Net      string `json:"net"`
	Address  string `json:"address"`
	TweetUrl string `json:"tweetUrl"`
}

type PreCheckReq struct {
	Net     string `json:"net"`
	Address string `json:"address"`
}
