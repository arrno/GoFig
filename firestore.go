package main

// TODO implement this
type Firestore interface {
	getDocData(docPath string) (map[string]any, error)
	genDocPath() (string, error)
	updateDoc(docPath string, data map[string]any) error
	setDoc(docPath string, data map[string]any) error
	deleteDoc(docPath string) error
}
