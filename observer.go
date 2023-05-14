package main

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/opts"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ReplyCmdOK  = "<CMD OK>"
	ReplyCmdERR = "<CMD Error>"

	CmdList = "list"
	CmdStat = "stat"
	CmdMem  = "mem"
	CmdCMem = "cmem"

	TimeoutKeyword = "TIMEOUT"
	ErrorKeyword   = "ERROR"
	TimeoutVal     = int64(0x3fffffff)
	ErrorVal       = TimeoutVal * 2

	KRED = "\x1B[31m"
	KNRM = "\x1B[0m"
)

const (
	ChartKeyCPU    = "CPU"
	ChartKeyMsgNum = "MSG_NUM"
	ChartKeyMqLen  = "MQ_LEN"
	ChartKeyLuaMem = "LUA_MEM"
	ChartKeyCMem   = "C_MEM"
)

var (
	ChartSubTitles = map[string]string{
		ChartKeyCPU:    "Y-axis unit: total ms",
		ChartKeyMsgNum: "Y-axis unit: total handled msg num",
		ChartKeyMqLen:  "Y-axis unit: msg queue len",
		ChartKeyLuaMem: "Y-axis unit: lua vm mem kb",
		ChartKeyCMem:   "Y-axis unit: c mem kb",
	}
)

type SamplingData struct {
	cpuTime int64 // ms
	msgNum  int64
	mqLen   int64
	luaMem  int64 // kb
	cMem    int64 // kb
}

func (sd *SamplingData) getValue(chartKey string) int64 {
	if chartKey == ChartKeyCPU {
		return sd.cpuTime
	} else if chartKey == ChartKeyMsgNum {
		return sd.msgNum
	} else if chartKey == ChartKeyMqLen {
		return sd.mqLen
	} else if chartKey == ChartKeyLuaMem {
		return sd.luaMem
	} else if chartKey == ChartKeyCMem {
		return sd.cMem
	} else {
		log.Fatalf("invalid chartKey: %s", chartKey)
	}
	return -1
}

type SamplingList []*SamplingData

func (sl SamplingList) getMaxValue(chartKey string) int64 {
	if len(sl) == 0 {
		return -1
	}
	maxVal := sl[0].getValue(chartKey)
	for _, sd := range sl {
		val := sd.getValue(chartKey)
		if maxVal < val {
			maxVal = val
		}
	}
	return maxVal
}

type Observer struct {
	ready   bool
	running bool
	round   int
	lastCmd string
	// key: <:0000000x>,	value: <snlua xxxx>
	svcLabels     map[string]string
	recvBuffer    string
	dialer        *Dialer
	samplingTimes []time.Time
	// key: <:0000000x>
	samplingDatas map[string]SamplingList
}

func (o *Observer) Init(address string, timeout time.Duration) {
	o.dialer = &Dialer{
		observer:  o,
		onMessage: o.onMessage,
	}
	o.dialer.Connect(address, timeout)
	o.SendCmd(CmdList)
}

func (o *Observer) Quit(reason ...string) {
	if len(reason) > 0 {
		log.Println(reason[0])
	}
	os.Exit(0)
}

func (o *Observer) Start() bool {
	if o.running {
		return false
	}
	o.samplingTimes = make([]time.Time, 0)
	o.samplingDatas = make(map[string]SamplingList)
	o.running = true
	o.round = 1
	o.SendCmd(CmdStat)
	o.samplingTimes = append(o.samplingTimes, time.Now())
	log.Println("----Start!!----")

	return true
}

func (o *Observer) Stop() bool {
	if !o.running {
		return false
	}
	o.running = false
	log.Println("----Stopped!!----")
	return true
}

func (o *Observer) SendCmd(s string) {
	if s != "" {
		o.dialer.Send([]byte(s + "\n"))
		o.lastCmd = s
	}
}

func (o *Observer) onMessage(b []byte) {
	sCmd := string(b)
	o.recvBuffer = o.recvBuffer + sCmd

	for {
		errIdx := strings.Index(o.recvBuffer, ReplyCmdERR)
		if errIdx >= 0 {
			out := o.recvBuffer[:errIdx]
			o.Quit(out)
		}

		idx := strings.Index(o.recvBuffer, ReplyCmdOK)
		if idx == -1 {
			break
		}

		out := o.recvBuffer[:idx]
		if !o.ready {
			o.onPrepare(out)
		} else {
			o.onReply(out)
		}

		o.recvBuffer = o.recvBuffer[idx+len(ReplyCmdOK):]
	}
}

func (o *Observer) onPrepare(out string) {
	if o.lastCmd != CmdList {
		o.Quit(fmt.Sprintf("prepare lastCmd != %s", CmdList))
	}

	lines := strings.Split(out, "\n")
	labels := make(map[string]string)
	for _, line := range lines {
		info := strings.Split(line, "\t")
		if len(info) == 2 {
			labels[info[0]] = info[1]
		}
	}
	o.svcLabels = labels
	o.ready = true
	log.Println("observer ready!")
}

func (o *Observer) onCmdStatReply(msg string) {
	var (
		svcName string
		cpuTime int64
		msgNum  int64
		mqLen   int64
	)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		info := strings.Split(line, "\t")
		if len(info) == 2 {
			if info[1] == TimeoutKeyword {
				svcName, cpuTime, msgNum, mqLen = info[0], TimeoutVal, TimeoutVal, TimeoutVal
			} else if info[1] == ErrorKeyword {
				svcName, cpuTime, msgNum, mqLen = info[0], ErrorVal, ErrorVal, ErrorVal
			} else {
				log.Fatalf("parse stat info line failed: %v", line)
			}
		} else if len(info) >= 5 {
			svcName, cpuTime, msgNum, mqLen = info[0], int64(ParseFloat(info[1], ":", 1)*1000+0.5), ParseInt(info[2], ":", 1), ParseInt(info[3], ":", 1)
		} else {
			continue
		}

		if o.samplingDatas[svcName] == nil {
			o.samplingDatas[svcName] = make([]*SamplingData, 0)
		}
		o.samplingDatas[svcName] = append(o.samplingDatas[svcName], &SamplingData{
			cpuTime: cpuTime,
			msgNum:  msgNum,
			mqLen:   mqLen,
			luaMem:  0,
			cMem:    0,
		})
	}
}

func (o *Observer) onCmdMemReply(msg string) {
	var (
		svcName string
		luaMem  int64
	)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		info := strings.Split(line, "\t")
		if len(info) != 2 {
			continue
		}
		svcName = info[0]
		if strings.HasPrefix(info[1], TimeoutKeyword) {
			luaMem = TimeoutVal
		} else if strings.HasPrefix(info[1], ErrorKeyword) {
			luaMem = ErrorVal
		} else {
			luaMem = int64(ParseFloat(info[1], " Kb", 0) + 0.5)
		}

		saDatas := o.samplingDatas[svcName]
		if len(saDatas) == 0 {
			continue
		}
		saDatas[len(saDatas)-1].luaMem = luaMem
	}
}

func (o *Observer) onCmdCMemReply(msg string) {
	var (
		svcName string
		cMem    int64
	)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		info := strings.Split(line, "\t")
		if len(info) != 2 {
			continue
		}
		svcName = info[0]
		if strings.HasPrefix(info[1], TimeoutKeyword) {
			cMem = TimeoutVal
		} else if strings.HasPrefix(info[1], ErrorKeyword) {
			cMem = ErrorVal
		} else {
			val, _ := strconv.ParseInt(info[1], 10, 64)
			cMem = int64(float64(val)/1000.0 + 0.5)
		}

		saDatas := o.samplingDatas[svcName]
		if len(saDatas) == 0 {
			continue
		}
		saDatas[len(saDatas)-1].cMem = cMem
	}
}

func (o *Observer) onReply(msg string) {
	if !o.running {
		return
	}
	if o.lastCmd == CmdStat {
		o.onCmdStatReply(msg)
		if o.running {
			o.SendCmd(CmdMem)
		}
	} else if o.lastCmd == CmdMem {
		o.onCmdMemReply(msg)
		if o.running {
			o.SendCmd(CmdCMem)
		}
	} else if o.lastCmd == CmdCMem {
		o.onCmdCMemReply(msg)
		log.Printf("sampling round %d done\n", o.round)
		o.round++
		if o.running {
			time.Sleep(time.Duration(samplingIntervalSec) * time.Second)
			if o.running {
				o.SendCmd(CmdStat)
				o.samplingTimes = append(o.samplingTimes, time.Now())
			}
		}
	}
}

func (o *Observer) fetchTopSvcs(chartKey string, topSvcNum int) []string {
	type SortItem struct {
		svcName string
		value   int64
	}
	itemList := make([]*SortItem, 0)
	for svcName, saDatas := range o.samplingDatas {
		if len(saDatas) == 0 {
			continue
		}
		itemList = append(itemList, &SortItem{
			svcName: svcName,
			value:   saDatas.getMaxValue(chartKey),
		})
	}

	sort.Slice(itemList, func(i, j int) bool {
		return itemList[i].value > itemList[j].value
	})

	topSvcs := make([]string, 0)
	for i, item := range itemList {
		if i >= topSvcNum {
			break
		}
		topSvcs = append(topSvcs, item.svcName)
	}
	return topSvcs
}

func (o *Observer) generateOneChart(chartKey string, topSvcNum int, displaySvcs map[string]string) *ChartData {
	XAxis := make([]string, 0)
	for _, t := range o.samplingTimes {
		XAxis = append(XAxis, t.Format("15:04:05"))
	}
	chartValue := &ChartData{
		XAxis: XAxis,
	}

	topSvcs := o.fetchTopSvcs(chartKey, topSvcNum)
	for i, svcName := range topSvcs {
		saDatas := o.samplingDatas[svcName]
		items := make([]opts.LineData, 0)
		for _, data := range saDatas {
			items = append(items, opts.LineData{Value: data.getValue(chartKey)})
		}
		suffix := ""
		maxVal := saDatas.getMaxValue(chartKey)
		if maxVal == TimeoutVal {
			suffix = "<timeout>"
		} else if maxVal == ErrorVal {
			suffix = "<error>"
		}
		chartValue.serieData = append(chartValue.serieData, &SerieData{
			name:  svcName + suffix,
			YAxis: items,
		})
		if displaySvcs != nil {
			svcDesc := fmt.Sprintf("%s:<%d>st ", chartKey, i+1)
			if i < 5 {
				svcDesc = KRED + svcDesc + KNRM
			}
			displaySvcs[svcName] += svcDesc
		}
	}

	return chartValue
}

func (o *Observer) dumpToChart(w http.ResponseWriter, topSvcNum int) {
	dumpList := []string{ChartKeyLuaMem, ChartKeyCMem, ChartKeyCPU, ChartKeyMsgNum, ChartKeyMqLen}
	displaySvcs := make(map[string]string)
	for _, key := range dumpList {
		line := NewChartLine(key, ChartSubTitles[key], o.generateOneChart(key, topSvcNum, displaySvcs))
		line.Render(w)
	}

	svcList := []string{}
	for svcName := range displaySvcs {
		svcList = append(svcList, svcName)
	}
	sort.Slice(svcList, func(i, j int) bool {
		return svcList[i] < svcList[j]
	})
	log.Printf("-----top %d services list, merge(%d)-----\n", topSvcNum, len(svcList))
	for _, svcName := range svcList {
		log.Printf("%s => %s (%s)\n", svcName, o.svcLabels[svcName], displaySvcs[svcName])
	}
}
