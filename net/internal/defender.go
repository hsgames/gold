package internal

import "time"

type Defender struct {
	lastMsgTime time.Time
	curMsgNum   int
	maxMsgNum   int
}

func NewDefender(maxMsgNum int) *Defender {
	return &Defender{
		maxMsgNum: maxMsgNum,
	}
}

func (l *Defender) CheckPacketSpeed() bool {
	if l.maxMsgNum <= 0 {
		return true
	}
	if now := time.Now(); now.Sub(l.lastMsgTime) > time.Second {
		l.lastMsgTime = now
		l.curMsgNum = 0
	}
	l.curMsgNum++
	return l.curMsgNum < l.maxMsgNum
}
