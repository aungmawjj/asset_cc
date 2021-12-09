package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating chaincode: %v", err)
	}

	if err := cc.Start(); err != nil {
		log.Panicf("Error starting chaincode: %v", err)
	}
}

type SmartContract struct {
	contractapi.Contract
}

const (
	KeyAssets        = "assets"
	KeyAuctions      = "auctions"
	KeyLastAuctionID = "lastAuction"
)

type Asset struct {
	ID               string
	Owner            string
	PendingAuctionID int
}

type Auction struct {
	ID              int
	AssetID         string
	Platforms       []string
	CrossAuctionIDs []string
	Status          string

	HighestBid         int
	HighestBidder      string
	HighestBidPlatform string
}

type StartAuctionArgs struct {
	AssetID   string
	Platforms []string
}

type BindAuctionArgs struct {
	AuctionID       int
	CrossAuctionIDs []string
}

type EndAuctionArgs struct {
	AuctionID      int
	HighestBids    []int
	HighestBidders []string
}

func (cc *SmartContract) AddAsset(
	ctx contractapi.TransactionContextInterface, id, owner string,
) error {
	asset := Asset{
		ID:    id,
		Owner: owner,
	}
	return cc.setAsset(ctx, &asset)
}

func (cc *SmartContract) StartAuction(
	ctx contractapi.TransactionContextInterface, argJSON string,
) error {
	var args StartAuctionArgs
	err := json.Unmarshal([]byte(argJSON), &args)
	if err != nil {
		return err
	}

	asset, err := cc.GetAsset(ctx, args.AssetID)
	if err != nil {
		return err
	}
	if asset.PendingAuctionID > 0 {
		return fmt.Errorf("pending auction on asset")
	}

	lastID, err := cc.GetLastAuctionID(ctx)
	if err != nil {
		return err
	}
	auction := Auction{
		ID:              lastID + 1,
		AssetID:         args.AssetID,
		Platforms:       args.Platforms,
		CrossAuctionIDs: make([]string, len(args.Platforms)),
		Status:          "Started",
	}
	err = cc.setAuction(ctx, &auction)
	if err != nil {
		return err
	}
	err = cc.setLastAuctionID(ctx, auction.ID)
	if err != nil {
		return err
	}

	asset.PendingAuctionID = auction.ID
	return cc.setAsset(ctx, asset)
}

func (cc *SmartContract) BindAuction(
	ctx contractapi.TransactionContextInterface, argJSON string,
) error {
	var args BindAuctionArgs
	err := json.Unmarshal([]byte(argJSON), &args)
	if err != nil {
		return err
	}
	auction, err := cc.GetAuction(ctx, args.AuctionID)
	if err != nil {
		return err
	}
	auction.Status = "Bind"
	auction.CrossAuctionIDs = args.CrossAuctionIDs
	return cc.setAuction(ctx, auction)
}

func (cc *SmartContract) SetAuctionEnding(
	ctx contractapi.TransactionContextInterface, assetID string,
) error {
	asset, err := cc.GetAsset(ctx, assetID)
	if err != nil {
		return err
	}
	auction, err := cc.GetAuction(ctx, asset.PendingAuctionID)
	if err != nil {
		return err
	}
	auction.Status = "Ending"
	return cc.setAuction(ctx, auction)
}
func (cc *SmartContract) EndAuction(
	ctx contractapi.TransactionContextInterface, argJSON string,
) error {

	var args EndAuctionArgs
	err := json.Unmarshal([]byte(argJSON), &args)
	if err != nil {
		return err
	}
	auction, err := cc.GetAuction(ctx, args.AuctionID)
	if err != nil {
		return err
	}

	for idx, bid := range args.HighestBids {
		if bid > auction.HighestBid {
			auction.HighestBid = bid
			auction.HighestBidPlatform = auction.Platforms[idx]
			auction.HighestBidder = args.HighestBidders[idx]
		}
	}

	auction.Status = "Ended"
	err = cc.setAuction(ctx, auction)
	if err != nil {
		return err
	}

	asset, err := cc.GetAsset(ctx, auction.AssetID)
	if err != nil {
		return err
	}

	asset.Owner = auction.HighestBidder
	asset.PendingAuctionID = 0
	err = cc.setAsset(ctx, asset)
	if err != nil {
		return err
	}
	return nil
}

func (cc *SmartContract) GetAsset(
	ctx contractapi.TransactionContextInterface, assetID string,
) (*Asset, error) {
	var asset Asset
	b, err := ctx.GetStub().GetState(cc.makeAssetKey(assetID))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("asset not found")
	}
	err = json.Unmarshal(b, &asset)
	return &asset, err
}

func (cc *SmartContract) GetAuction(
	ctx contractapi.TransactionContextInterface, auctionID int,
) (*Auction, error) {
	b, err := ctx.GetStub().GetState(cc.makeAuctionKey(auctionID))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("auction not found")
	}
	var auction Auction
	err = json.Unmarshal(b, &auction)
	return &auction, err
}

func (cc *SmartContract) GetLastAuctionID(
	ctx contractapi.TransactionContextInterface,
) (int, error) {
	b, err := ctx.GetStub().GetState(KeyLastAuctionID)
	if err != nil {
		return 0, err
	}
	var count int
	json.Unmarshal(b, &count)
	return count, nil
}

func (cc *SmartContract) setAsset(
	ctx contractapi.TransactionContextInterface, asset *Asset,
) error {
	b, _ := json.Marshal(asset)
	err := ctx.GetStub().PutState(cc.makeAssetKey(asset.ID), b)
	if err != nil {
		return fmt.Errorf("set asset error: %v", err)
	}
	return nil
}

func (cc *SmartContract) setAuction(
	ctx contractapi.TransactionContextInterface, auction *Auction,
) error {
	b, _ := json.Marshal(auction)
	err := ctx.GetStub().PutState(cc.makeAuctionKey(auction.ID), b)
	if err != nil {
		return fmt.Errorf("set auction error: %v", err)
	}
	return nil
}

func (cc *SmartContract) setLastAuctionID(
	ctx contractapi.TransactionContextInterface, id int,
) error {
	b, _ := json.Marshal(id)
	return ctx.GetStub().PutState(KeyLastAuctionID, b)
}

func (cc *SmartContract) makeAssetKey(assetID string) string {
	return fmt.Sprintf("%s_%s", KeyAssets, assetID)
}

func (cc *SmartContract) makeAuctionKey(auctionID int) string {
	return fmt.Sprintf("%s_%d", KeyAuctions, auctionID)
}
