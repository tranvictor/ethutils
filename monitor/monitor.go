package monitor

import (
	"sync"
	"time"

	eu "github.com/tranvictor/ethutils"
	"github.com/tranvictor/ethutils/reader"
)

type TxMonitor struct {
	reader *reader.EthReader
}

func NewGenericTxMonitor(r *reader.EthReader) *TxMonitor {
	return &TxMonitor{r}
}

func NewRopstenTxMonitor() *TxMonitor {
	return &TxMonitor{
		reader: reader.NewRopstenReader(),
	}
}

func NewTxMonitor() *TxMonitor {
	return &TxMonitor{
		reader: reader.NewEthReader(),
	}
}

func (self TxMonitor) periodicCheck(tx string, info chan eu.TxInfo) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	startTime := time.Now()
	isOnNode := false
	for {
		t := <-ticker.C
		txinfo, _ := self.reader.TxInfoFromHash(tx)
		st, tx, receipt := txinfo.Status, txinfo.Tx, txinfo.Receipt
		switch st {
		case "error":
			continue
		case "notfound":
			if t.Sub(startTime) > 3*time.Minute && !isOnNode {
				info <- eu.TxInfo{"lost", tx, []eu.InternalTx{}, receipt}
				return
			} else {
				continue
			}
		case "pending":
			isOnNode = true
			continue
		case "reverted":
			info <- eu.TxInfo{"reverted", tx, []eu.InternalTx{}, receipt}
			return
		case "done":
			info <- eu.TxInfo{"done", tx, []eu.InternalTx{}, receipt}
			return
		}
	}
}

func (self TxMonitor) MakeWaitChannel(tx string) <-chan eu.TxInfo {
	result := make(chan eu.TxInfo)
	go self.periodicCheck(tx, result)
	return result
}

func (self TxMonitor) BlockingWait(tx string) eu.TxInfo {
	wChannel := self.MakeWaitChannel(tx)
	return <-wChannel
}

func (self TxMonitor) MakeWaitChannelForMultipleTxs(txs ...string) []<-chan eu.TxInfo {
	result := [](<-chan eu.TxInfo){}
	for _, tx := range txs {
		ch := make(chan eu.TxInfo)
		go self.periodicCheck(tx, ch)
		result = append(result, ch)
	}
	return result
}

func waitForChannel(wg *sync.WaitGroup, channel <-chan eu.TxInfo, result *sync.Map) {
	defer wg.Done()
	info := <-channel
	result.Store(info.Tx.Hash().Hex(), info)
}

func (self TxMonitor) BlockingWaitForMultipleTxs(txs ...string) map[string]eu.TxInfo {
	resultMap := sync.Map{}
	wg := sync.WaitGroup{}
	channels := self.MakeWaitChannelForMultipleTxs(txs...)
	for _, channel := range channels {
		wg.Add(1)
		go waitForChannel(&wg, channel, &resultMap)
	}
	wg.Wait()
	result := map[string]eu.TxInfo{}
	resultMap.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(eu.TxInfo)
		return true
	})
	return result
}
