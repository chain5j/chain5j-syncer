// Package syncer
//
// @author: xwc1125
package syncer

func (s *syncer) findAncestor(peer *peer, remoteHeight uint64) {
	var (
		localHeight uint64
	)
	localHeight = s.blockRW.CurrentBlock().Height()

	from, count, skip, _ := calculateRequestSpan(remoteHeight, localHeight)
	go s.RequestHeadersByNumber(peer.P2PID, uint64(from), count, skip, false)
}

// common ancestor.
// It returns parameters to be used for peer.RequestHeadersByNumber:
//  from - starting block number
//  count - number of headers to request
//  skip - number of headers to skip
// and also returns 'max', the last block which is expected to be returned by the remote peers,
// given the (from,count,skip)
func calculateRequestSpan(remoteHeight, localHeight uint64) (int64, int, int, uint64) {
	var (
		from     int
		count    int
		MaxCount = MaxHeaderFetch
	)
	// requestHead is the highest block that we will ask for. If requestHead is not offset,
	// the highest block that we will get is 16 blocks back from head, which means we
	// will fetch 14 or 15 blocks unnecessarily in the case the height difference
	// between us and the peer is 1-2 blocks, which is most common
	requestHead := int(remoteHeight) - 1
	if requestHead < 0 {
		requestHead = 0
	}
	// requestBottom is the lowest block we want included in the query
	// Ideally, we want to include just below own head
	requestBottom := int(localHeight - 1)
	if requestBottom < 0 {
		requestBottom = 0
	}

	totalSpan := requestHead - requestBottom
	count = MaxCount
	if totalSpan < MaxCount {
		count = totalSpan
	}

	span := 1 + totalSpan/MaxCount
	if span < 2 {
		span = 2
	}
	if span > 16 {
		span = 16
	}

	// count = 1 + totalSpan/span
	// if count > MaxCount {
	//	count = MaxCount
	// }
	// if count < 2 {
	//	count = 2
	// }
	// from = requestHead - (count-1)*span
	// if from < 0 {
	//	from = 0
	// }
	from = int(localHeight + 1)
	max := from + count
	return int64(from), count, span - 1, uint64(max)
}

// func calculateRequestSpan(remoteHeight, localHeight uint64) (int64, int, int, uint64) {
//	var (
//		from  int
//		count int
//		//MaxCount = MaxHeaderFetch / 16 // max count=12
//		MaxCount = MaxHeaderFetch
//	)
//	// requestHead is the highest block that we will ask for. If requestHead is not offset,
//	// the highest block that we will get is 16 blocks back from head, which means we
//	// will fetch 14 or 15 blocks unnecessarily in the case the height difference
//	// between us and the peer is 1-2 blocks, which is most common
//	requestHead := int(remoteHeight) - 1
//	if requestHead < 0 {
//		requestHead = 0
//	}
//	// requestBottom is the lowest block we want included in the query
//	// Ideally, we want to include just below own head
//	requestBottom := int(localHeight - 1)
//	if requestBottom < 0 {
//		requestBottom = 0
//	}
//	totalSpan := requestHead - requestBottom
//	span := 1 + totalSpan/MaxCount
//	if span < 2 {
//		span = 2
//	}
//	if span > 16 {
//		span = 16
//	}
//
//	count = 1 + totalSpan/span
//	if count > MaxCount {
//		count = MaxCount
//	}
//	if count < 2 {
//		count = 2
//	}
//	//from = requestHead - (count-1)*span
//	//if from < 0 {
//	//	from = 0
//	//}
//	from = int(localHeight + 1)
//	max := from + (count-1)*span
//	return int64(from), count, span - 1, uint64(max)
// }
