package fbconn

// consumeInlineBlob returns cached inline blob data for the transaction and removes it from the cache.
// When tx is nil, the internal transaction is used.
func (fbConn *FbConn) consumeInlineBlob(tx *FBTransaction, blobId int64) ([]byte, bool) {
	if fbConn == nil || fbConn.client == nil {
		return nil, false
	}
	if tx == nil {
		tx = fbConn.internalTransaction
	}
	if tx == nil || tx.Closed {
		return nil, false
	}
	return fbConn.client.ConsumeInlineBlob(tx.Handle, blobId)
}

func (fbConn *FbConn) clearInlineBlobCache(txHandle int) {
	if fbConn == nil || fbConn.client == nil {
		return
	}
	fbConn.client.ClearInlineBlobCache(txHandle)
}

func (fbConn *FbConn) clearAllInlineBlobCache() {
	if fbConn == nil || fbConn.client == nil {
		return
	}
	fbConn.client.ClearAllInlineBlobCache()
}

// releaseInlineBlobData returns inline blob data to the pool (if enabled).
func (fbConn *FbConn) releaseInlineBlobData(data []byte) {
	if fbConn == nil || fbConn.client == nil || data == nil {
		return
	}
	fbConn.client.ReleaseInlineBlobData(data)
}
