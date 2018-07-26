package message

type BlockHeightRequestMessage struct {
	Height uint64
}

type BlockHeightResponseMessage struct {
	Height uint64
}

type BlockRequestMessage struct {
	Height uint64
}

type BlockResponseMessage struct {
}
