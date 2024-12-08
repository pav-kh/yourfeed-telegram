package pool

import (
	"sync"

	"github.com/pav-kh/yourfeed-telegram/internal/models"
)

// ObjectPool manages reusable objects to reduce GC pressure
type ObjectPool struct {
	messagePool sync.Pool // Pool for MessageTask objects
	bufferPool  sync.Pool // Pool for byte buffers
}

// NewObjectPool creates a new ObjectPool instance
func NewObjectPool() *ObjectPool {
	return &ObjectPool{
		messagePool: sync.Pool{
			New: func() interface{} {
				return &models.MessageTask{}
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
	}
}

// GetMessageTask gets a MessageTask from the pool
func (p *ObjectPool) GetMessageTask() *models.MessageTask {
	return p.messagePool.Get().(*models.MessageTask)
}

// PutMessageTask returns a MessageTask to the pool
func (p *ObjectPool) PutMessageTask(task *models.MessageTask) {
	p.messagePool.Put(task)
}

// GetBuffer gets a byte buffer from the pool
func (p *ObjectPool) GetBuffer() []byte {
	return p.bufferPool.Get().([]byte)
}

// PutBuffer returns a byte buffer to the pool
func (p *ObjectPool) PutBuffer(buf []byte) {
	p.bufferPool.Put(buf[:0])
}
