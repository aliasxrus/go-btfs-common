package main

import (
	"context"

	"github.com/tron-us/go-btfs-common/crypto"
	"github.com/tron-us/go-btfs-common/ledger"
	ledgerpb "github.com/tron-us/go-btfs-common/protos/ledger"
	"github.com/tron-us/go-btfs-common/utils/grpc"
	"github.com/tron-us/go-common/v2/log"

	"go.uber.org/zap"
)

const (
	PayerPrivKeyString    = "CAISIJFNZZd5ZSvi9OlJP/mz/vvUobvlrr2//QN4DzX/EShP"
	ReceiverPrivKeyString = "CAISIDm/qF5f98Jh8FGBUcFUhQvJPU8uEah1SZrR1BrGekC0"
)

func main() {
	err := grpc.LedgerClient("https://ledger-dev.bt.co:443").WithContext(context.Background(),
		func(ctx context.Context, ledgerClient ledgerpb.ChannelsClient) error {
			// create payer Account
			payerPrivKey, err := crypto.ToPrivKey(PayerPrivKeyString)
			if err != nil {
				log.Panic("can not convert to private key", zap.Error(err))
			}
			payerPubKey := payerPrivKey.GetPublic()
			_, err = ledger.ImportSignedAccount(ctx, payerPrivKey, payerPubKey, ledgerClient)
			if err != nil {
				log.Panic("can not create account on ledger", zap.Error(err))
			}
			// create receiver account
			recvPrivKey, err := crypto.ToPrivKey(ReceiverPrivKeyString)
			if err != nil {
				log.Panic("can not convert to private key", zap.Error(err))
			}
			recvPubKey := recvPrivKey.GetPublic()
			_, err = ledger.ImportSignedAccount(ctx, recvPrivKey, recvPubKey, ledgerClient)
			if err != nil {
				log.Panic("can not create account on ledger", zap.Error(err))
			}
			// prepare channel commit
			amount := int64(1)
			channelCommit, err := ledger.NewChannelCommit(payerPubKey, recvPubKey, amount)
			if err != nil {
				log.Panic("can not create channel commit", zap.Error(err))
			}
			// sign for the channel commit
			fromSig, err := crypto.Sign(payerPrivKey, channelCommit)
			if err != nil {
				log.Panic("fail to sign channel commit", zap.Error(err))
			}
			// create channel: payer start the channel
			channelID, err := ledger.CreateChannel(ctx, ledgerClient, channelCommit, fromSig)
			if err != nil {
				log.Panic("fail to create channel", zap.Error(err))
			}
			// channel state: transfer money from -> to
			fromAcc, err := ledger.NewAccount(payerPubKey, 0)
			if err != nil {
				log.Panic("wrong account on channel", zap.Error(err))
			}
			toAcc, err := ledger.NewAccount(recvPubKey, amount)
			if err != nil {
				log.Panic("wrong account on channel", zap.Error(err))
			}
			channelState := ledger.NewChannelState(channelID, 1, fromAcc, toAcc)
			// need permission from both account, get signature from both
			fromSigState, err := crypto.Sign(payerPrivKey, channelState)
			if err != nil {
				log.Panic("error when signing the channel state", zap.Error(err))
			}
			toSigState, err := crypto.Sign(recvPrivKey, channelState)
			if err != nil {
				log.Panic("error when signing the channel state", zap.Error(err))
			}
			signedChannelState := ledger.NewSignedChannelState(channelState, fromSigState, toSigState)
			// close channel
			err = ledger.CloseChannel(ctx, ledgerClient, signedChannelState)
			if err != nil {
				log.Panic("fail to close channel", zap.Error(err))
			}
			return nil
		})
	if err != nil {
		log.Panic(err.Error())
	}
}
