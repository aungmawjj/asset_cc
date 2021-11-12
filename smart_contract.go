package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating asset chaincode: %v", err)
	}

	if err := cc.Start(); err != nil {
		log.Panicf("Error starting asset chaincode: %v", err)
	}
}

type SmartContract struct {
	contractapi.Contract
	ctx contractapi.TransactionContextInterface
}

const (
	KeyAssets = "assets"
)

type Asset struct {
	ID             []byte
	Owner          []byte
	PendingAuction *Auction
}

type Auction struct {
	ID       []byte
	Platform string
}

type AuctionResult struct {
	Auction
	HighestBidder []byte
}

type BindAuctionArgs struct {
	AssetID []byte
	Auction Auction
}

type EndAuctionArgs struct {
	AssetID       []byte
	AuctionResult AuctionResult
}

func (cc *SmartContract) AddAsset(ctx contractapi.TransactionContextInterface, arg string) error {
	cc.ctx = ctx
	var asset Asset
	err := json.Unmarshal([]byte(arg), &asset)
	if err != nil {
		return err
	}
	return cc.setAsset(asset)
}

func (cc *SmartContract) BindAuction(ctx contractapi.TransactionContextInterface, arg string) error {
	cc.ctx = ctx
	var args BindAuctionArgs
	err := json.Unmarshal([]byte(arg), &args)
	if err != nil {
		return err
	}
	asset, err := cc.getAsset(args.AssetID)
	if err != nil {
		return err
	}
	asset.PendingAuction = &args.Auction
	return cc.setAsset(asset)
}

func (cc *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface, arg string) error {
	cc.ctx = ctx
	var args EndAuctionArgs
	err := json.Unmarshal([]byte(arg), &args)
	if err != nil {
		return err
	}
	asset, err := cc.getAsset(args.AssetID)
	if err != nil {
		return err
	}
	if asset.PendingAuction == nil {
		return fmt.Errorf("no pending auction")
	}
	if !bytes.Equal(asset.PendingAuction.ID, args.AuctionResult.ID) {
		return fmt.Errorf("invalid auction result")
	}

	// transfer asset to winner
	asset.Owner = args.AuctionResult.HighestBidder
	asset.PendingAuction = nil
	return cc.setAsset(asset)
}

func (cc *SmartContract) GetAsset(
	ctx contractapi.TransactionContextInterface, arg string,
) ([]byte, error) {
	cc.ctx = ctx
	assetID, err := base64.StdEncoding.DecodeString(arg)
	if err != nil {
		return nil, err
	}
	return ctx.GetStub().GetState(cc.makeAssetKey(assetID))
}

func (cc *SmartContract) getAsset(assetID []byte) (Asset, error) {
	var asset Asset
	b, err := cc.ctx.GetStub().GetState(cc.makeAssetKey(assetID))
	if err != nil {
		return asset, err
	}
	if b == nil {
		return asset, fmt.Errorf("asset not found")
	}
	err = json.Unmarshal(b, &asset)
	return asset, err
}

func (cc *SmartContract) setAsset(asset Asset) error {
	b, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	return cc.ctx.GetStub().PutState(cc.makeAssetKey(asset.ID), b)
}

func (cc *SmartContract) makeAssetKey(assetID []byte) string {
	return fmt.Sprintf("%s_%s", KeyAssets, assetID)
}