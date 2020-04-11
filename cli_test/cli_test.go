// +build cli_test

package clitest

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/stretchr/testify/require"

	"github.com/shinecloudfoundation/shinecloudnet/app"

	"github.com/shinecloudfoundation/shinecloudnet/client"
	"github.com/shinecloudfoundation/shinecloudnet/tests"
	sdk "github.com/shinecloudfoundation/shinecloudnet/types"
	"github.com/shinecloudfoundation/shinecloudnet/x/auth"
	"github.com/shinecloudfoundation/shinecloudnet/x/genaccounts"
	"github.com/shinecloudfoundation/shinecloudnet/x/gov"
	"github.com/shinecloudfoundation/shinecloudnet/x/mint"
)

func TestScloudCLIKeysAddMultisig(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// key names order does not matter
	f.KeysAdd("msig1", "--multisig-threshold=2",
		fmt.Sprintf("--multisig=%s,%s", keyBar, keyBaz))
	f.KeysAdd("msig2", "--multisig-threshold=2",
		fmt.Sprintf("--multisig=%s,%s", keyBaz, keyBar))
	require.Equal(t, f.KeysShow("msig1").Address, f.KeysShow("msig2").Address)

	f.KeysAdd("msig3", "--multisig-threshold=2",
		fmt.Sprintf("--multisig=%s,%s", keyBar, keyBaz),
		"--nosort")
	f.KeysAdd("msig4", "--multisig-threshold=2",
		fmt.Sprintf("--multisig=%s,%s", keyBaz, keyBar),
		"--nosort")
	require.NotEqual(t, f.KeysShow("msig3").Address, f.KeysShow("msig4").Address)

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIKeysAddRecover(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	exitSuccess, _, _ := f.KeysAddRecover("empty-mnemonic", "")
	require.False(t, exitSuccess)

	exitSuccess, _, _ = f.KeysAddRecover("test-recover", "dentist task convince chimney quality leave banana trade firm crawl eternal easily")
	require.True(t, exitSuccess)
	require.Equal(t, "scloud1qcfdf69js922qrdr4yaww3ax7gjml6pd2pw7nf", f.KeyAddress("test-recover").String())

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIKeysAddRecoverHDPath(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	f.KeysAddRecoverHDPath("test-recoverHD1", "dentist task convince chimney quality leave banana trade firm crawl eternal easily", 0, 0)
	require.Equal(t, "scloud1qcfdf69js922qrdr4yaww3ax7gjml6pd2pw7nf", f.KeyAddress("test-recoverHD1").String())

	f.KeysAddRecoverHDPath("test-recoverH2", "dentist task convince chimney quality leave banana trade firm crawl eternal easily", 1, 5)
	require.Equal(t, "scloud1pdfav2cjhry9k79nu6r8kgknnjtq6a7rr8qenc", f.KeyAddress("test-recoverH2").String())

	f.KeysAddRecoverHDPath("test-recoverH3", "dentist task convince chimney quality leave banana trade firm crawl eternal easily", 1, 17)
	require.Equal(t, "scloud1909k354n6wl8ujzu6kmh49w4d02ax7qvc8h320", f.KeyAddress("test-recoverH3").String())

	f.KeysAddRecoverHDPath("test-recoverH4", "dentist task convince chimney quality leave banana trade firm crawl eternal easily", 2, 17)
	require.Equal(t, "scloud1v9plmhvyhgxk3th9ydacm7j4z357s3nhvltk8h", f.KeyAddress("test-recoverH4").String())

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIMinimumFees(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	minGasPrice, _ := sdk.NewDecFromStr("0.000006")
	fees := fmt.Sprintf(
		"--minimum-gas-prices=%s,%s",
		sdk.NewDecCoinFromDec(feeDenom, minGasPrice),
		sdk.NewDecCoinFromDec(fee2Denom, minGasPrice),
	)
	proc := f.GDStart(fees)
	defer proc.Stop(false)

	barAddr := f.KeyAddress(keyBar)

	// Send a transaction that will get rejected
	success, stdOut, _ := f.TxSend(keyFoo, barAddr, sdk.NewInt64Coin(fee2Denom, 10), "-y")
	require.Contains(t, stdOut, "insufficient fees")
	require.True(f.T, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure tx w/ correct fees pass
	txFees := fmt.Sprintf("--fees=%s", sdk.NewInt64Coin(feeDenom, 2))
	success, _, _ = f.TxSend(keyFoo, barAddr, sdk.NewInt64Coin(fee2Denom, 10), txFees, "-y")
	require.True(f.T, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure tx w/ improper fees fails
	txFees = fmt.Sprintf("--fees=%s", sdk.NewInt64Coin(feeDenom, 1))
	success, _, _ = f.TxSend(keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, 10), txFees, "-y")
	require.Contains(t, stdOut, "insufficient fees")
	require.True(f.T, success)

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIGasPrices(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	minGasPrice, _ := sdk.NewDecFromStr("0.000006")
	proc := f.GDStart(fmt.Sprintf("--minimum-gas-prices=%s", sdk.NewDecCoinFromDec(feeDenom, minGasPrice)))
	defer proc.Stop(false)

	barAddr := f.KeyAddress(keyBar)

	// insufficient gas prices (tx fails)
	badGasPrice, _ := sdk.NewDecFromStr("0.000003")
	success, stdOut, _ := f.TxSend(
		keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, 50),
		fmt.Sprintf("--gas-prices=%s", sdk.NewDecCoinFromDec(feeDenom, badGasPrice)), "-y")
	require.Contains(t, stdOut, "insufficient fees")
	require.True(t, success)

	// wait for a block confirmation
	tests.WaitForNextNBlocksTM(1, f.Port)

	// sufficient gas prices (tx passes)
	success, _, _ = f.TxSend(
		keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, 50),
		fmt.Sprintf("--gas-prices=%s", sdk.NewDecCoinFromDec(feeDenom, minGasPrice)), "-y")
	require.True(t, success)

	// wait for a block confirmation
	tests.WaitForNextNBlocksTM(1, f.Port)

	f.Cleanup()
}

func TestScloudCLIFeesDeduction(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	minGasPrice, _ := sdk.NewDecFromStr("0.000006")
	proc := f.GDStart(fmt.Sprintf("--minimum-gas-prices=%s", sdk.NewDecCoinFromDec(feeDenom, minGasPrice)))
	defer proc.Stop(false)

	// Save key addresses for later use
	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	fooAcc := f.QueryAccount(fooAddr)
	fooAmt := fooAcc.GetCoins().AmountOf(fooDenom)

	// test simulation
	success, _, _ := f.TxSend(
		keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, 1000),
		fmt.Sprintf("--fees=%s", sdk.NewInt64Coin(feeDenom, 2)), "--dry-run")
	require.True(t, success)

	// Wait for a block
	tests.WaitForNextNBlocksTM(1, f.Port)

	// ensure state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, fooAmt.Int64(), fooAcc.GetCoins().AmountOf(fooDenom).Int64())

	// insufficient funds (coins + fees) tx fails
	largeCoins := sdk.TokensFromConsensusPower(10000000)
	success, stdOut, _ := f.TxSend(
		keyFoo, barAddr, sdk.NewCoin(fooDenom, largeCoins),
		fmt.Sprintf("--fees=%s", sdk.NewInt64Coin(feeDenom, 2)), "-y")
	require.Contains(t, stdOut, "insufficient account funds")
	require.True(t, success)

	// Wait for a block
	tests.WaitForNextNBlocksTM(1, f.Port)

	// ensure state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, fooAmt.Int64(), fooAcc.GetCoins().AmountOf(fooDenom).Int64())

	// test success (transfer = coins + fees)
	success, _, _ = f.TxSend(
		keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, 500),
		fmt.Sprintf("--fees=%s", sdk.NewInt64Coin(feeDenom, 2)), "-y")
	require.True(t, success)

	f.Cleanup()
}

func TestScloudCLISend(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	// Save key addresses for later use
	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Send some tokens from one account to the other
	sendTokens := sdk.TokensFromConsensusPower(10)
	f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account balances match expected
	barAcc := f.QueryAccount(barAddr)
	require.Equal(t, sendTokens, barAcc.GetCoins().AmountOf(denom))
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(sendTokens), fooAcc.GetCoins().AmountOf(denom))

	// Test --dry-run
	success, _, _ := f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--dry-run")
	require.True(t, success)

	// Test --generate-only
	success, stdout, stderr := f.TxSend(
		fooAddr.String(), barAddr, sdk.NewCoin(denom, sendTokens), "--generate-only=true",
	)
	require.Empty(t, stderr)
	require.True(t, success)
	msg := unmarshalStdTx(f.T, stdout)
	require.NotZero(t, msg.Fee.Gas)
	require.Len(t, msg.Msgs, 1)
	require.Len(t, msg.GetSignatures(), 0)

	// Check state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(sendTokens), fooAcc.GetCoins().AmountOf(denom))

	// test autosequencing
	f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account balances match expected
	barAcc = f.QueryAccount(barAddr)
	require.Equal(t, sendTokens.MulRaw(2), barAcc.GetCoins().AmountOf(denom))
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(sendTokens.MulRaw(2)), fooAcc.GetCoins().AmountOf(denom))

	// test memo
	f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--memo='testmemo'", "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account balances match expected
	barAcc = f.QueryAccount(barAddr)
	require.Equal(t, sendTokens.MulRaw(3), barAcc.GetCoins().AmountOf(denom))
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(sendTokens.MulRaw(3)), fooAcc.GetCoins().AmountOf(denom))

	f.Cleanup()
}

func TestScloudCLIGasAuto(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Test failure with auto gas disabled and very little gas set by hand
	sendTokens := sdk.TokensFromConsensusPower(10)
	success, stdOut, _ := f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--gas=10", "-y")
	require.Contains(t, stdOut, "out of gas in location")
	require.True(t, success)

	// Check state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Test failure with negative gas
	success, _, _ = f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--gas=-100", "-y")
	require.False(t, success)

	// Check state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Test failure with 0 gas
	success, stdOut, _ = f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--gas=0", "-y")
	require.Contains(t, stdOut, "out of gas in location")
	require.True(t, success)

	// Check state didn't change
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Enable auto gas
	success, stdout, stderr := f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "--gas=auto", "-y")
	require.NotEmpty(t, stderr)
	require.True(t, success)
	cdc := app.MakeCodec()
	sendResp := sdk.TxResponse{}
	err := cdc.UnmarshalJSON([]byte(stdout), &sendResp)
	require.Nil(t, err)
	require.True(t, sendResp.GasWanted >= sendResp.GasUsed)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Check state has changed accordingly
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(sendTokens), fooAcc.GetCoins().AmountOf(denom))

	f.Cleanup()
}

func TestScloudCLICreateValidator(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	barAddr := f.KeyAddress(keyBar)
	barVal := sdk.ValAddress(barAddr)

	consPubKey := sdk.MustBech32ifyConsPub(ed25519.GenPrivKey().PubKey())

	sendTokens := sdk.TokensFromConsensusPower(10)
	f.TxSend(keyFoo, barAddr, sdk.NewCoin(denom, sendTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	barAcc := f.QueryAccount(barAddr)
	require.Equal(t, sendTokens, barAcc.GetCoins().AmountOf(denom))

	// Generate a create validator transaction and ensure correctness
	success, stdout, stderr := f.TxStakingCreateValidator(barAddr.String(), consPubKey, sdk.NewInt64Coin(denom, 2), "--generate-only")

	require.True(f.T, success)
	require.Empty(f.T, stderr)
	msg := unmarshalStdTx(f.T, stdout)
	require.NotZero(t, msg.Fee.Gas)
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Test --dry-run
	newValTokens := sdk.TokensFromConsensusPower(2)
	success, _, _ = f.TxStakingCreateValidator(keyBar, consPubKey, sdk.NewCoin(denom, newValTokens), "--dry-run")
	require.True(t, success)

	// Create the validator
	f.TxStakingCreateValidator(keyBar, consPubKey, sdk.NewCoin(denom, newValTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure funds were deducted properly
	barAcc = f.QueryAccount(barAddr)
	require.Equal(t, sendTokens.Sub(newValTokens), barAcc.GetCoins().AmountOf(denom))

	// Ensure that validator state is as expected
	validator := f.QueryStakingValidator(barVal)
	require.Equal(t, validator.OperatorAddress, barVal)
	require.True(sdk.IntEq(t, newValTokens, validator.Tokens))

	// Query delegations to the validator
	validatorDelegations := f.QueryStakingDelegationsTo(barVal)
	require.Len(t, validatorDelegations, 1)
	require.NotZero(t, validatorDelegations[0].Shares)

	// unbond a single share
	unbondAmt := sdk.NewCoin(sdk.DefaultBondDenom, sdk.TokensFromConsensusPower(1))
	success = f.TxStakingUnbond(keyBar, unbondAmt.String(), barVal, "-y")
	require.True(t, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure bonded staking is correct
	remainingTokens := newValTokens.Sub(unbondAmt.Amount)
	validator = f.QueryStakingValidator(barVal)
	require.Equal(t, remainingTokens, validator.Tokens)

	// Get unbonding delegations from the validator
	validatorUbds := f.QueryStakingUnbondingDelegationsFrom(barVal)
	require.Len(t, validatorUbds, 1)
	require.Len(t, validatorUbds[0].Entries, 1)
	require.Equal(t, remainingTokens.String(), validatorUbds[0].Entries[0].Balance.String())

	f.Cleanup()
}

func TestScloudCLIQueryRewards(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)
	cdc := app.MakeCodec()

	genesisState := f.GenesisState()
	var mintData mint.GenesisState
	cdc.UnmarshalJSON(genesisState[mint.ModuleName], &mintData)
	mintData.Minter.RemainedTokens = nil
	mintData.Params.MintDenom = sdk.DefaultBondDenom
	mintData.Params.UnfreezeAmountPerBlock = 1000000
	mintDataBz, err := cdc.MarshalJSON(mintData)
	require.NoError(t, err)
	genesisState[mint.ModuleName] = mintDataBz

	genFile := filepath.Join(f.ScloudHome, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	genDoc.AppState, err = cdc.MarshalJSON(genesisState)
	require.NoError(t, genDoc.SaveAs(genFile))

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	rewards := f.QueryRewards(fooAddr)
	require.Equal(t, 1, len(rewards.Rewards))

	f.Cleanup()
}

func TestScloudCLIQuerySupply(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	totalSupply := f.QueryTotalSupply()
	totalSupplyOf := f.QueryTotalSupplyOf(fooDenom)

	require.Equal(t, totalCoins, totalSupply)
	require.True(sdk.IntEq(t, totalCoins.AmountOf(fooDenom), totalSupplyOf))

	f.Cleanup()
}

func TestScloudCLISubmitProposal(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	f.QueryGovParamDeposit()
	f.QueryGovParamVoting()
	f.QueryGovParamTallying()

	fooAddr := f.KeyAddress(keyFoo)

	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(sdk.DefaultBondDenom))

	proposalsQuery := f.QueryGovProposals()
	require.Empty(t, proposalsQuery)

	// Test submit generate only for submit proposal
	proposalTokens := sdk.TokensFromConsensusPower(5)
	success, stdout, stderr := f.TxGovSubmitProposal(
		fooAddr.String(), "Text", "Test", "test", sdk.NewCoin(denom, proposalTokens), "--generate-only", "-y")
	require.True(t, success)
	require.Empty(t, stderr)
	msg := unmarshalStdTx(t, stdout)
	require.NotZero(t, msg.Fee.Gas)
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Test --dry-run
	success, _, _ = f.TxGovSubmitProposal(keyFoo, "Text", "Test", "test", sdk.NewCoin(denom, proposalTokens), "--dry-run")
	require.True(t, success)

	// Create the proposal
	f.TxGovSubmitProposal(keyFoo, "Text", "Test", "test", sdk.NewCoin(denom, proposalTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure transaction tags can be queried
	searchResult := f.QueryTxs(1, 50, "message.action:submit_proposal", fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, searchResult.Txs, 1)

	// Ensure deposit was deducted
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(proposalTokens), fooAcc.GetCoins().AmountOf(denom))

	// Ensure propsal is directly queryable
	proposal1 := f.QueryGovProposal(1)
	require.Equal(t, uint64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusDepositPeriod, proposal1.Status)

	// Ensure query proposals returns properly
	proposalsQuery = f.QueryGovProposals()
	require.Equal(t, uint64(1), proposalsQuery[0].ProposalID)

	// Query the deposits on the proposal
	deposit := f.QueryGovDeposit(1, fooAddr)
	require.Equal(t, proposalTokens, deposit.Amount.AmountOf(denom))

	// Test deposit generate only
	depositTokens := sdk.TokensFromConsensusPower(10)
	success, stdout, stderr = f.TxGovDeposit(1, fooAddr.String(), sdk.NewCoin(denom, depositTokens), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)
	msg = unmarshalStdTx(t, stdout)
	require.NotZero(t, msg.Fee.Gas)
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Run the deposit transaction
	f.TxGovDeposit(1, keyFoo, sdk.NewCoin(denom, depositTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// test query deposit
	deposits := f.QueryGovDeposits(1)
	require.Len(t, deposits, 1)
	require.Equal(t, proposalTokens.Add(depositTokens), deposits[0].Amount.AmountOf(denom))

	// Ensure querying the deposit returns the proper amount
	deposit = f.QueryGovDeposit(1, fooAddr)
	require.Equal(t, proposalTokens.Add(depositTokens), deposit.Amount.AmountOf(denom))

	// Ensure tags are set on the transaction
	searchResult = f.QueryTxs(1, 50, "message.action:deposit", fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, searchResult.Txs, 1)

	// Ensure account has expected amount of funds
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(proposalTokens.Add(depositTokens)), fooAcc.GetCoins().AmountOf(denom))

	// Fetch the proposal and ensure it is now in the voting period
	proposal1 = f.QueryGovProposal(1)
	require.Equal(t, uint64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusVotingPeriod, proposal1.Status)

	// Test vote generate only
	success, stdout, stderr = f.TxGovVote(1, gov.OptionYes, fooAddr.String(), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)
	msg = unmarshalStdTx(t, stdout)
	require.NotZero(t, msg.Fee.Gas)
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Vote on the proposal
	f.TxGovVote(1, gov.OptionYes, keyFoo, "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Query the vote
	vote := f.QueryGovVote(1, fooAddr)
	require.Equal(t, uint64(1), vote.ProposalID)
	require.Equal(t, gov.OptionYes, vote.Option)

	// Query the votes
	votes := f.QueryGovVotes(1)
	require.Len(t, votes, 1)
	require.Equal(t, uint64(1), votes[0].ProposalID)
	require.Equal(t, gov.OptionYes, votes[0].Option)

	// Ensure tags are applied to voting transaction properly
	searchResult = f.QueryTxs(1, 50, "message.action:vote", fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, searchResult.Txs, 1)

	// Ensure no proposals in deposit period
	proposalsQuery = f.QueryGovProposals("--status=DepositPeriod")
	require.Empty(t, proposalsQuery)

	// Ensure the proposal returns as in the voting period
	proposalsQuery = f.QueryGovProposals("--status=VotingPeriod")
	require.Equal(t, uint64(1), proposalsQuery[0].ProposalID)

	// submit a second test proposal
	f.TxGovSubmitProposal(keyFoo, "Text", "Apples", "test", sdk.NewCoin(denom, proposalTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Test limit on proposals query
	proposalsQuery = f.QueryGovProposals("--limit=1")
	require.Equal(t, uint64(2), proposalsQuery[0].ProposalID)

	f.Cleanup()
}

func TestScloudCLISubmitParamChangeProposal(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(sdk.DefaultBondDenom))

	// write proposal to file
	proposalTokens := sdk.TokensFromConsensusPower(5)
	proposal := fmt.Sprintf(`{
  "title": "Param Change",
  "description": "Update max validators",
  "changes": [
    {
      "subspace": "staking",
      "key": "MaxValidators",
      "value": 105
    }
  ],
  "deposit": [
    {
      "denom": "uscds",
      "amount": "%s"
    }
  ]
}
`, proposalTokens.String())

	proposalFile := WriteToNewTempFile(t, proposal)

	// create the param change proposal
	f.TxGovSubmitParamChangeProposal(keyFoo, proposalFile.Name(), sdk.NewCoin(denom, proposalTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// ensure transaction tags can be queried
	txsPage := f.QueryTxs(1, 50, "message.action:submit_proposal", fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage.Txs, 1)

	// ensure deposit was deducted
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(proposalTokens).String(), fooAcc.GetCoins().AmountOf(sdk.DefaultBondDenom).String())

	// ensure proposal is directly queryable
	proposal1 := f.QueryGovProposal(1)
	require.Equal(t, uint64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusDepositPeriod, proposal1.Status)

	// ensure correct query proposals result
	proposalsQuery := f.QueryGovProposals()
	require.Equal(t, uint64(1), proposalsQuery[0].ProposalID)

	// ensure the correct deposit amount on the proposal
	deposit := f.QueryGovDeposit(1, fooAddr)
	require.Equal(t, proposalTokens, deposit.Amount.AmountOf(denom))

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLISubmitCommunityPoolSpendProposal(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// create some inflation
	cdc := app.MakeCodec()
	genesisState := f.GenesisState()
	var mintData mint.GenesisState
	cdc.UnmarshalJSON(genesisState[mint.ModuleName], &mintData)
	mintData.Minter.RemainedTokens = nil
	mintData.Params.MintDenom = sdk.DefaultBondDenom
	mintData.Params.UnfreezeAmountPerBlock = 1000000
	mintDataBz, err := cdc.MarshalJSON(mintData)
	require.NoError(t, err)
	genesisState[mint.ModuleName] = mintDataBz

	genFile := filepath.Join(f.ScloudHome, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	genDoc.AppState, err = cdc.MarshalJSON(genesisState)
	require.NoError(t, genDoc.SaveAs(genFile))

	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(sdk.DefaultBondDenom))

	tests.WaitForNextNBlocksTM(3, f.Port)

	// write proposal to file
	proposalTokens := sdk.TokensFromConsensusPower(5)
	proposal := fmt.Sprintf(`{
  "title": "Community Pool Spend",
  "description": "Spend from community pool",
  "recipient": "%s",
  "amount": [
    {
      "denom": "%s",
      "amount": "1"
    }
  ],
  "deposit": [
    {
      "denom": "%s",
      "amount": "%s"
    }
  ]
}
`, fooAddr, sdk.DefaultBondDenom, sdk.DefaultBondDenom, proposalTokens.String())
	proposalFile := WriteToNewTempFile(t, proposal)

	// create the param change proposal
	f.TxGovSubmitCommunityPoolSpendProposal(keyFoo, proposalFile.Name(), sdk.NewCoin(denom, proposalTokens), "-y")
	tests.WaitForNextNBlocksTM(1, f.Port)

	// ensure transaction tags can be queried
	txsPage := f.QueryTxs(1, 50, "message.action:submit_proposal", fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage.Txs, 1)

	// ensure deposit was deducted
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, startTokens.Sub(proposalTokens).String(), fooAcc.GetCoins().AmountOf(sdk.DefaultBondDenom).String())

	// ensure proposal is directly queryable
	proposal1 := f.QueryGovProposal(1)
	require.Equal(t, uint64(1), proposal1.ProposalID)
	require.Equal(t, gov.StatusDepositPeriod, proposal1.Status)

	// ensure correct query proposals result
	proposalsQuery := f.QueryGovProposals()
	require.Equal(t, uint64(1), proposalsQuery[0].ProposalID)

	// ensure the correct deposit amount on the proposal
	deposit := f.QueryGovDeposit(1, fooAddr)
	require.Equal(t, proposalTokens, deposit.Amount.AmountOf(denom))

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIQueryTxPagination(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	accFoo := f.QueryAccount(fooAddr)
	seq := accFoo.GetSequence()

	for i := 1; i <= 30; i++ {
		success, _, _ := f.TxSend(keyFoo, barAddr, sdk.NewInt64Coin(fooDenom, int64(i)), fmt.Sprintf("--sequence=%d", seq), "-y")
		require.True(t, success)
		seq++
	}

	// perPage = 15, 2 pages
	txsPage1 := f.QueryTxs(1, 15, fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage1.Txs, 15)
	require.Equal(t, txsPage1.Count, 15)
	txsPage2 := f.QueryTxs(2, 15, fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage2.Txs, 15)
	require.NotEqual(t, txsPage1.Txs, txsPage2.Txs)

	// perPage = 16, 2 pages
	txsPage1 = f.QueryTxs(1, 16, fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage1.Txs, 16)
	txsPage2 = f.QueryTxs(2, 16, fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPage2.Txs, 14)
	require.NotEqual(t, txsPage1.Txs, txsPage2.Txs)

	// perPage = 50
	txsPageFull := f.QueryTxs(1, 50, fmt.Sprintf("message.sender:%s", fooAddr))
	require.Len(t, txsPageFull.Txs, 30)
	require.Equal(t, txsPageFull.Txs, append(txsPage1.Txs, txsPage2.Txs...))

	// perPage = 0
	f.QueryTxsInvalid(errors.New("ERROR: page must greater than 0"), 0, 50, fmt.Sprintf("message.sender:%s", fooAddr))

	// limit = 0
	f.QueryTxsInvalid(errors.New("ERROR: limit must greater than 0"), 1, 0, fmt.Sprintf("message.sender:%s", fooAddr))

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIValidateSignatures(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	// generate sendTx with default gas
	success, stdout, stderr := f.TxSend(fooAddr.String(), barAddr, sdk.NewInt64Coin(denom, 10), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)

	// write  unsigned tx to file
	unsignedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(unsignedTxFile.Name())

	// validate we can successfully sign
	success, stdout, stderr = f.TxSign(keyFoo, unsignedTxFile.Name())
	require.True(t, success)
	require.Empty(t, stderr)
	stdTx := unmarshalStdTx(t, stdout)
	require.Equal(t, len(stdTx.Msgs), 1)
	require.Equal(t, 1, len(stdTx.GetSignatures()))
	require.Equal(t, fooAddr.String(), stdTx.GetSigners()[0].String())

	// write signed tx to file
	signedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(signedTxFile.Name())

	// validate signatures
	success, _, _ = f.TxSign(keyFoo, signedTxFile.Name(), "--validate-signatures")
	require.True(t, success)

	// modify the transaction
	stdTx.Memo = "MODIFIED-ORIGINAL-TX-BAD"
	bz := marshalStdTx(t, stdTx)
	modSignedTxFile := WriteToNewTempFile(t, string(bz))
	defer os.Remove(modSignedTxFile.Name())

	// validate signature validation failure due to different transaction sig bytes
	success, _, _ = f.TxSign(keyFoo, modSignedTxFile.Name(), "--validate-signatures")
	require.False(t, success)

	f.Cleanup()
}

func TestScloudCLISendGenerateSignAndBroadcast(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	fooAddr := f.KeyAddress(keyFoo)
	barAddr := f.KeyAddress(keyBar)

	// Test generate sendTx with default gas
	sendTokens := sdk.TokensFromConsensusPower(10)
	success, stdout, stderr := f.TxSend(fooAddr.String(), barAddr, sdk.NewCoin(denom, sendTokens), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)
	msg := unmarshalStdTx(t, stdout)
	require.Equal(t, msg.Fee.Gas, uint64(client.DefaultGasLimit))
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Test generate sendTx with --gas=$amount
	success, stdout, stderr = f.TxSend(fooAddr.String(), barAddr, sdk.NewCoin(denom, sendTokens), "--gas=100", "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)
	msg = unmarshalStdTx(t, stdout)
	require.Equal(t, msg.Fee.Gas, uint64(100))
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 0, len(msg.GetSignatures()))

	// Test generate sendTx, estimate gas
	success, stdout, stderr = f.TxSend(fooAddr.String(), barAddr, sdk.NewCoin(denom, sendTokens), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)
	msg = unmarshalStdTx(t, stdout)
	require.True(t, msg.Fee.Gas > 0)
	require.Equal(t, len(msg.Msgs), 1)

	// Write the output to disk
	unsignedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(unsignedTxFile.Name())

	// Test sign --validate-signatures
	success, stdout, _ = f.TxSign(keyFoo, unsignedTxFile.Name(), "--validate-signatures")
	require.False(t, success)
	require.Equal(t, fmt.Sprintf("Signers:\n  0: %v\n\nSignatures:\n\n", fooAddr.String()), stdout)

	// Test sign
	success, stdout, _ = f.TxSign(keyFoo, unsignedTxFile.Name())
	require.True(t, success)
	msg = unmarshalStdTx(t, stdout)
	require.Equal(t, len(msg.Msgs), 1)
	require.Equal(t, 1, len(msg.GetSignatures()))
	require.Equal(t, fooAddr.String(), msg.GetSigners()[0].String())

	// Write the output to disk
	signedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(signedTxFile.Name())

	// Test sign --validate-signatures
	success, stdout, _ = f.TxSign(keyFoo, signedTxFile.Name(), "--validate-signatures")
	require.True(t, success)
	require.Equal(t, fmt.Sprintf("Signers:\n  0: %v\n\nSignatures:\n  0: %v\t\t\t[OK]\n\n", fooAddr.String(),
		fooAddr.String()), stdout)

	// Ensure foo has right amount of funds
	fooAcc := f.QueryAccount(fooAddr)
	startTokens := sdk.TokensFromConsensusPower(50)
	require.Equal(t, startTokens, fooAcc.GetCoins().AmountOf(denom))

	// Test broadcast
	success, stdout, _ = f.TxBroadcast(signedTxFile.Name())
	require.True(t, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account state
	barAcc := f.QueryAccount(barAddr)
	fooAcc = f.QueryAccount(fooAddr)
	require.Equal(t, sendTokens, barAcc.GetCoins().AmountOf(denom))
	require.Equal(t, startTokens.Sub(sendTokens), fooAcc.GetCoins().AmountOf(denom))

	f.Cleanup()
}

func TestScloudCLIMultisignInsufficientCosigners(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	proc := f.GDStart()
	defer proc.Stop(false)

	fooBarBazAddr := f.KeyAddress(keyFooBarBaz)
	barAddr := f.KeyAddress(keyBar)

	// Send some tokens from one account to the other
	success, _, _ := f.TxSend(keyFoo, fooBarBazAddr, sdk.NewInt64Coin(denom, 10), "-y")
	require.True(t, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Test generate sendTx with multisig
	success, stdout, _ := f.TxSend(fooBarBazAddr.String(), barAddr, sdk.NewInt64Coin(denom, 5), "--generate-only")
	require.True(t, success)

	// Write the output to disk
	unsignedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(unsignedTxFile.Name())

	// Sign with foo's key
	success, stdout, _ = f.TxSign(keyFoo, unsignedTxFile.Name(), "--multisig", fooBarBazAddr.String(), "-y")
	require.True(t, success)

	// Write the output to disk
	fooSignatureFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(fooSignatureFile.Name())

	// Multisign, not enough signatures
	success, stdout, _ = f.TxMultisign(unsignedTxFile.Name(), keyFooBarBaz, []string{fooSignatureFile.Name()})
	require.True(t, success)

	// Write the output to disk
	signedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(signedTxFile.Name())

	// Validate the multisignature
	success, _, _ = f.TxSign(keyFooBarBaz, signedTxFile.Name(), "--validate-signatures")
	require.False(t, success)

	// Broadcast the transaction
	success, stdOut, _ := f.TxBroadcast(signedTxFile.Name())
	require.Contains(t, stdOut, "signature verification failed")
	require.True(t, success)

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIEncode(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	cdc := app.MakeCodec()

	// Build a testing transaction and write it to disk
	barAddr := f.KeyAddress(keyBar)
	keyAddr := f.KeyAddress(keyFoo)

	sendTokens := sdk.TokensFromConsensusPower(10)
	success, stdout, stderr := f.TxSend(keyAddr.String(), barAddr, sdk.NewCoin(denom, sendTokens), "--generate-only", "--memo", "deadbeef")
	require.True(t, success)
	require.Empty(t, stderr)

	// Write it to disk
	jsonTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(jsonTxFile.Name())

	// Run the encode command, and trim the extras from the stdout capture
	success, base64Encoded, _ := f.TxEncode(jsonTxFile.Name())
	require.True(t, success)
	trimmedBase64 := strings.Trim(base64Encoded, "\"\n")

	// Decode the base64
	decodedBytes, err := base64.StdEncoding.DecodeString(trimmedBase64)
	require.Nil(t, err)

	// Check that the transaction decodes as epxceted
	var decodedTx auth.StdTx
	require.Nil(t, cdc.UnmarshalBinaryLengthPrefixed(decodedBytes, &decodedTx))
	require.Equal(t, "deadbeef", decodedTx.Memo)
}

func TestScloudCLIMultisignSortSignatures(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	proc := f.GDStart()
	defer proc.Stop(false)

	fooBarBazAddr := f.KeyAddress(keyFooBarBaz)
	barAddr := f.KeyAddress(keyBar)

	// Send some tokens from one account to the other
	success, _, _ := f.TxSend(keyFoo, fooBarBazAddr, sdk.NewInt64Coin(denom, 10), "-y")
	require.True(t, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account balances match expected
	fooBarBazAcc := f.QueryAccount(fooBarBazAddr)
	require.Equal(t, int64(10), fooBarBazAcc.GetCoins().AmountOf(denom).Int64())

	// Test generate sendTx with multisig
	success, stdout, _ := f.TxSend(fooBarBazAddr.String(), barAddr, sdk.NewInt64Coin(denom, 5), "--generate-only")
	require.True(t, success)

	// Write the output to disk
	unsignedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(unsignedTxFile.Name())

	// Sign with foo's key
	success, stdout, _ = f.TxSign(keyFoo, unsignedTxFile.Name(), "--multisig", fooBarBazAddr.String())
	require.True(t, success)

	// Write the output to disk
	fooSignatureFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(fooSignatureFile.Name())

	// Sign with baz's key
	success, stdout, _ = f.TxSign(keyBaz, unsignedTxFile.Name(), "--multisig", fooBarBazAddr.String())
	require.True(t, success)

	// Write the output to disk
	bazSignatureFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(bazSignatureFile.Name())

	// Multisign, keys in different order
	success, stdout, _ = f.TxMultisign(unsignedTxFile.Name(), keyFooBarBaz, []string{
		bazSignatureFile.Name(), fooSignatureFile.Name()})
	require.True(t, success)

	// Write the output to disk
	signedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(signedTxFile.Name())

	// Validate the multisignature
	success, _, _ = f.TxSign(keyFooBarBaz, signedTxFile.Name(), "--validate-signatures")
	require.True(t, success)

	// Broadcast the transaction
	success, _, _ = f.TxBroadcast(signedTxFile.Name())
	require.True(t, success)

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIMultisign(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server with minimum fees
	proc := f.GDStart()
	defer proc.Stop(false)

	fooBarBazAddr := f.KeyAddress(keyFooBarBaz)
	bazAddr := f.KeyAddress(keyBaz)

	// Send some tokens from one account to the other
	success, _, _ := f.TxSend(keyFoo, fooBarBazAddr, sdk.NewInt64Coin(denom, 10), "-y")
	require.True(t, success)
	tests.WaitForNextNBlocksTM(1, f.Port)

	// Ensure account balances match expected
	fooBarBazAcc := f.QueryAccount(fooBarBazAddr)
	require.Equal(t, int64(10), fooBarBazAcc.GetCoins().AmountOf(denom).Int64())

	// Test generate sendTx with multisig
	success, stdout, stderr := f.TxSend(fooBarBazAddr.String(), bazAddr, sdk.NewInt64Coin(denom, 10), "--generate-only")
	require.True(t, success)
	require.Empty(t, stderr)

	// Write the output to disk
	unsignedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(unsignedTxFile.Name())

	// Sign with foo's key
	success, stdout, _ = f.TxSign(keyFoo, unsignedTxFile.Name(), "--multisig", fooBarBazAddr.String(), "-y")
	require.True(t, success)

	// Write the output to disk
	fooSignatureFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(fooSignatureFile.Name())

	// Sign with bar's key
	success, stdout, _ = f.TxSign(keyBar, unsignedTxFile.Name(), "--multisig", fooBarBazAddr.String(), "-y")
	require.True(t, success)

	// Write the output to disk
	barSignatureFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(barSignatureFile.Name())

	// Multisign
	success, stdout, _ = f.TxMultisign(unsignedTxFile.Name(), keyFooBarBaz, []string{
		fooSignatureFile.Name(), barSignatureFile.Name()})
	require.True(t, success)

	// Write the output to disk
	signedTxFile := WriteToNewTempFile(t, stdout)
	defer os.Remove(signedTxFile.Name())

	// Validate the multisignature
	success, _, _ = f.TxSign(keyFooBarBaz, signedTxFile.Name(), "--validate-signatures", "-y")
	require.True(t, success)

	// Broadcast the transaction
	success, _, _ = f.TxBroadcast(signedTxFile.Name())
	require.True(t, success)

	// Cleanup testing directories
	f.Cleanup()
}

func TestScloudCLIConfig(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)
	node := fmt.Sprintf("%s:%s", f.RPCAddr, f.Port)

	// Set available configuration options
	f.CLIConfig("broadcast-mode", "block")
	f.CLIConfig("node", node)
	f.CLIConfig("output", "text")
	f.CLIConfig("trust-node", "true")
	f.CLIConfig("chain-id", f.ChainID)
	f.CLIConfig("trace", "false")
	f.CLIConfig("indent", "true")

	config, err := ioutil.ReadFile(path.Join(f.ScloudcliHome, "config", "config.toml"))
	require.NoError(t, err)
	expectedConfig := fmt.Sprintf(`broadcast-mode = "block"
chain-id = "%s"
indent = true
node = "%s"
output = "text"
trace = false
trust-node = true
`, f.ChainID, node)
	require.Equal(t, expectedConfig, string(config))

	f.Cleanup()
}

func TestScloudCollectGentxs(t *testing.T) {
	t.Parallel()
	var customMaxBytes, customMaxGas int64 = 99999999, 1234567
	f := NewFixtures(t)

	// Initialise temporary directories
	gentxDir, err := ioutil.TempDir("", "")
	gentxDoc := filepath.Join(gentxDir, "gentx.json")
	require.NoError(t, err)

	// Reset testing path
	f.UnsafeResetAll()

	// Initialize keys
	f.KeysAdd(keyFoo)

	// Configure json output
	f.CLIConfig("output", "json")

	// Run init
	f.GDInit(keyFoo)

	// Customise genesis.json

	genFile := f.GenesisFile()
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	genDoc.ConsensusParams.Block.MaxBytes = customMaxBytes
	genDoc.ConsensusParams.Block.MaxGas = customMaxGas
	genDoc.SaveAs(genFile)

	// Add account to genesis.json
	f.AddGenesisAccount(f.KeyAddress(keyFoo), startCoins)

	// Write gentx file
	f.GenTx(keyFoo, fmt.Sprintf("--output-document=%s", gentxDoc))

	// Collect gentxs from a custom directory
	f.CollectGenTxs(fmt.Sprintf("--gentx-dir=%s", gentxDir))

	genDoc, err = tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	require.Equal(t, genDoc.ConsensusParams.Block.MaxBytes, customMaxBytes)
	require.Equal(t, genDoc.ConsensusParams.Block.MaxGas, customMaxGas)

	f.Cleanup(gentxDir)
}

func TestScloudAddGenesisAccount(t *testing.T) {
	t.Parallel()
	f := NewFixtures(t)

	// Reset testing path
	f.UnsafeResetAll()

	// Initialize keys
	f.KeysDelete(keyFoo)
	f.KeysDelete(keyBar)
	f.KeysDelete(keyBaz)
	f.KeysAdd(keyFoo)
	f.KeysAdd(keyBar)
	f.KeysAdd(keyBaz)

	// Configure json output
	f.CLIConfig("output", "json")

	// Run init
	f.GDInit(keyFoo)

	// Add account to genesis.json
	bazCoins := sdk.Coins{
		sdk.NewInt64Coin("acoin", 1000000),
		sdk.NewInt64Coin("bcoin", 1000000),
	}

	f.AddGenesisAccount(f.KeyAddress(keyFoo), startCoins)
	f.AddGenesisAccount(f.KeyAddress(keyBar), bazCoins)
	genesisState := f.GenesisState()

	cdc := app.MakeCodec()
	accounts := genaccounts.GetGenesisStateFromAppState(cdc, genesisState)

	require.Equal(t, accounts[0].Address, f.KeyAddress(keyFoo))
	require.Equal(t, accounts[1].Address, f.KeyAddress(keyBar))
	require.True(t, accounts[0].Coins.IsEqual(startCoins))
	require.True(t, accounts[1].Coins.IsEqual(bazCoins))

	// Cleanup testing directories
	f.Cleanup()
}

func TestSlashingGetParams(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	params := f.QuerySlashingParams()
	require.Equal(t, time.Duration(120000000000), params.MaxEvidenceAge)
	require.Equal(t, int64(100), params.SignedBlocksWindow)
	require.Equal(t, sdk.NewDecWithPrec(5, 1), params.MinSignedPerWindow)

	sinfo := f.QuerySigningInfo(f.GDTendermint("show-validator"))
	require.Equal(t, int64(0), sinfo.StartHeight)
	require.False(t, sinfo.Tombstoned)

	// Cleanup testing directories
	f.Cleanup()
}

func TestValidateGenesis(t *testing.T) {
	t.Parallel()
	f := InitFixtures(t)

	// start scloud server
	proc := f.GDStart()
	defer proc.Stop(false)

	f.ValidateGenesis()

	// Cleanup testing directories
	f.Cleanup()
}
