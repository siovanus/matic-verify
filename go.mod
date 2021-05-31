module github.com/zhiqiangxu/matic-verify

go 1.15

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4

replace github.com/tendermint/tendermint => github.com/maticnetwork/tendermint v0.26.0-dev0.0.20210419165356-d045b7a4e8fd

replace github.com/cosmos/cosmos-sdk => github.com/maticnetwork/cosmos-sdk v0.37.5-0.20210419165708-5d75f0b3ea99

require (
	github.com/maticnetwork/heimdall v0.3.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/tendermint/tendermint v0.34.0
)
