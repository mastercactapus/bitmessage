package bitmessage

import (
	"github.com/boltdb/bolt"
	"os"
)

type Store interface {
	SaveObject(InvVector, []byte) error
	GetObject(InvVector) ([]byte, error)
	ListObjects() ([]InvVector, error)
	Close() error
}

var objectBucket = []byte("object_storage")

type FileStore struct {
	db *bolt.DB
}

func NewFileStore(file string, mode os.FileMode) (*FileStore, error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(objectBucket)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &FileStore{db: db}, nil
}

func (fs *FileStore) Close() error {
	return fs.db.Close()
}

func (fs *FileStore) SaveObject(v InvVector, data []byte) error {
	return fs.db.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists(objectBucket)
		if err != nil {
			return err
		}
		return bk.Put(v[:], data)
	})
}
func (fs *FileStore) GetObject(v InvVector) ([]byte, error) {
	var data []byte
	err := fs.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket(objectBucket)
		data = bk.Get(v[:])
		return nil
	})
	return data, err
}
func (fs *FileStore) ListObjects() ([]InvVector, error) {
	var result []InvVector
	err := fs.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket(objectBucket)
		result = make([]InvVector, 0, bk.Stats().KeyN)
		return bk.ForEach(func(k, v []byte) error {
			var vc InvVector
			copy(vc[:], k)
			result = append(result, vc)
			return nil
		})
	})
	return result, err
}
