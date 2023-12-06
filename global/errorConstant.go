package global

const (
	// success
	SUCCESS    int    = 200
	SUCCESSMsg string = "success"

	// Common Error
	ParseErrCode int    = 100000
	ParseErrMsg  string = "Params parse err"

	CommonErrCode int    = 100001
	CommonErrMsg  string = "Axiomledger-Aries Network Error，Please Try Again Later！"

	// Business Error
	ErrAddrCode int    = 110000
	ErrAddrMsg  string = "Invalid address: "

	NotSupportCode int    = 110001
	NotSupportMsg  string = "Not support net: "

	ReqWithinDayCode int    = 110002
	ReqWithinDayMsg  string = "Sorry! To be fair to all developers, we only send 100 AXM every 24 hours. Please try again after 24 hours from your original request."

	EnoughTokenCode int    = 110003
	EnoughTokenMsg  string = "The address already has enough test tokens"

	InsufficientCode int    = 110004
	InsufficientMsg  string = "Faucet error"

	TweetAddrErrCode int    = 110005
	TweetAddrErrMsg  string = "The address in the tweet does not match the currently filled in address"

	TweetLinkErrCode int    = 110006
	TweetLinkErrMsg  string = "Tweet does not meet requirements, link is missing"

	TweetTimeErrCode int    = 110007
	TweetTimeErrMsg  string = "Tweet does not meet requirements, this tweet was sent 24 hours ago."

	TweetUrlErrCode int    = 110008
	TweetUrlErrMsg  string = "Invalid address tweet link"

	AddrPreLockErrCode int    = 110009
	AddrPreLockErrMsg  string = "The account is still being processed"

	// BlockChain Error
	BlockChainCode int    = 120000
	BlockChainMsg  string = "Blockchain communication error"

	// Scrapper Error
	ScrapperErrCode int    = 130000
	ScrapperErrMsg  string = "Someting went wrong, please try again later."
)
