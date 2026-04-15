//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && gc_opt

package xnet

import (
	"sync/atomic"

	"github.com/xbaseio/xbase/utils/xgfd"
)

// connMatrix 是连接矩阵，用于按二维表结构管理连接。
type connMatrix struct {
	disableCompact bool                           // 为 true 时禁用压缩整理
	connCounts     [xgfd.ConnMatrixRowMax]int32   // 每一行的活跃连接数
	row            int                            // 下一个可用行索引
	column         int                            // 下一个可用列索引
	table          [xgfd.ConnMatrixRowMax][]*conn // 连接矩阵
	fd2gfd         map[int]xgfd.GFD               // fd -> xgfd.GFD 的映射
}

// init 初始化连接矩阵内部结构。
func (cm *connMatrix) init() {
	cm.fd2gfd = make(map[int]xgfd.GFD)
}

// iterate 遍历矩阵中的所有连接。
// 遍历期间会临时禁用压缩整理，避免遍历过程中结构被移动。
func (cm *connMatrix) iterate(f func(*conn) bool) {
	cm.disableCompact = true
	defer func() {
		cm.disableCompact = false
	}()

	for _, conns := range cm.table {
		for _, c := range conns {
			if c != nil {
				if !f(c) {
					return
				}
			}
		}
	}
}

// incCount 增加或减少指定行的连接计数。
func (cm *connMatrix) incCount(row int, delta int32) {
	atomic.AddInt32(&cm.connCounts[row], delta)
}

// loadCount 统计当前矩阵中的总连接数。
func (cm *connMatrix) loadCount() (n int32) {
	for i := 0; i < len(cm.connCounts); i++ {
		n += atomic.LoadInt32(&cm.connCounts[i])
	}
	return
}

// addConn 将连接添加到矩阵中。
func (cm *connMatrix) addConn(c *conn, index int) {
	if cm.row >= xgfd.ConnMatrixRowMax {
		return
	}

	if cm.table[cm.row] == nil {
		cm.table[cm.row] = make([]*conn, xgfd.ConnMatrixColumnMax)
	}

	c.xgfd = xgfd.NewGFD(c.fd, index, cm.row, cm.column)
	cm.fd2gfd[c.fd] = c.xgfd
	cm.table[cm.row][cm.column] = c
	cm.incCount(cm.row, 1)

	cm.column++
	if cm.column == xgfd.ConnMatrixColumnMax {
		cm.row++
		cm.column = 0
	}
}

// delConn 从矩阵中删除连接。
// 删除后会尝试将矩阵尾部的最后一个连接移动到当前空位，以保持结构紧凑。
func (cm *connMatrix) delConn(c *conn) {
	cfd, cgfd := c.fd, c.xgfd

	delete(cm.fd2gfd, cfd)
	cm.incCount(cgfd.ConnMatrixRow(), -1)

	if cm.connCounts[cgfd.ConnMatrixRow()] == 0 {
		cm.table[cgfd.ConnMatrixRow()] = nil
	} else {
		cm.table[cgfd.ConnMatrixRow()][cgfd.ConnMatrixColumn()] = nil
	}

	if cm.row > cgfd.ConnMatrixRow() || cm.column > cgfd.ConnMatrixColumn() {
		cm.row, cm.column = cgfd.ConnMatrixRow(), cgfd.ConnMatrixColumn()
	}

	// 找到矩阵中的最后一个连接，并移动到当前删除位置。

	// 如果当前禁用了压缩，或者被删除的连接本来就是最后一个连接，则无需处理。
	if cm.disableCompact || cm.table[cgfd.ConnMatrixRow()] == nil {
		return
	}

	// 从后往前查找第一个非空位置，直到回溯到被删除的位置。
	for row := xgfd.ConnMatrixRowMax - 1; row >= cgfd.ConnMatrixRow(); row-- {
		if cm.connCounts[row] == 0 {
			continue
		}

		columnMin := -1
		if row == cgfd.ConnMatrixRow() {
			columnMin = cgfd.ConnMatrixColumn()
		}

		for column := xgfd.ConnMatrixColumnMax - 1; column > columnMin; column-- {
			if cm.table[row][column] == nil {
				continue
			}

			gfdVal := cm.table[row][column].xgfd
			gfdVal.UpdateIndexes(cgfd.ConnMatrixRow(), cgfd.ConnMatrixColumn())

			cm.table[row][column].xgfd = gfdVal
			cm.fd2gfd[gfdVal.Fd()] = gfdVal
			cm.table[cgfd.ConnMatrixRow()][cgfd.ConnMatrixColumn()] = cm.table[row][column]

			cm.incCount(row, -1)
			cm.incCount(cgfd.ConnMatrixRow(), 1)

			if cm.connCounts[row] == 0 {
				cm.table[row] = nil
			} else {
				cm.table[row][column] = nil
			}

			cm.row, cm.column = row, column
			return
		}
	}
}

// getConn 根据 fd 获取连接。
func (cm *connMatrix) getConn(fd int) *conn {
	gfdVal, ok := cm.fd2gfd[fd]
	if !ok {
		return nil
	}
	if cm.table[gfdVal.ConnMatrixRow()] == nil {
		return nil
	}
	return cm.table[gfdVal.ConnMatrixRow()][gfdVal.ConnMatrixColumn()]
}

/*
func (cm *connMatrix) getConnByGFD(fd xgfd.GFD) *conn {
	if cm.table[fd.ConnMatrixRow()] == nil {
		return nil
	}
	return cm.table[fd.ConnMatrixRow()][fd.ConnMatrixColumn()]
}
*/
