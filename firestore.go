package fig

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/aidarkhanov/nanoid"
	"google.golang.org/api/option"
)

// figFirestore is an interface that expresses what a NoSQL database dependency should do.
type figFirestore interface {
	getDocData(docPath string) (map[string]any, error)
	genDocPath(colPath string) (string, error)
	updateDoc(docPath string, data map[string]any) error
	setDoc(docPath string, data map[string]any) error
	deleteDoc(docPath string) error
	deleteField() any
	refField(docPath string) any
	name() string
}

// fireFriend is the gofig implementation/wrapper for google's firestore client.
type fireFriend struct {
	client *firestore.Client
	ctx    context.Context
	config map[string]string
}

// newFirestore is a fireFriend factory
func newFirestore(keyPath string) (*fireFriend, func(), error) {

	ctx := context.Background()

	sa := option.WithCredentialsFile(keyPath)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, func() {}, err
	}

	client, err := app.Firestore(ctx)
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	alphabet += strings.ToUpper(alphabet) + "0123456789_-"
	config := map[string]string{
		"idChars": alphabet,
		"idSize":  "20",
	}

	f := fireFriend{
		client,
		ctx,
		config,
	}
	return &f, func() { client.Close() }, err

}

// name returns a hash for the underlying firestore database name. This may
// be useful for guarding against pushing changes to the wrong database.
func (f fireFriend) name() string {
	s := strings.Split(f.client.Doc("__init__/__init__").Path, "__init__/__init__")[0]
	return strings.Split(s, "/databases/(default)/documents/")[0]
}

// docRef converts a string path to a firestore document reference.
func (f fireFriend) docRef(path string) (*firestore.DocumentRef, error) {

	ref := f.client.Doc(path)

	if ref == nil {
		return ref, errors.New("Unable to create doc reference.")
	}

	return ref, nil

}

// doc converts a string path to a firestore document snapshot.
func (f fireFriend) doc(path string) (*firestore.DocumentSnapshot, error) {

	ref, err := f.docRef(path)

	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(f.ctx)

	return snap, err
}

// getDocData attempts to read the specified document. If the document exists, it returns the underlying data.
func (f fireFriend) getDocData(docPath string) (map[string]any, error) {

	snap, err := f.doc(docPath)
	if !snap.Exists() {
		return map[string]any{}, nil
	}

	if err != nil {
		return map[string]interface{}{}, err

	}
	return snap.Data(), nil
}

// genDocPath is used to generate new unique document path given a collection path.
func (f fireFriend) genDocPath(colPath string) (string, error) {

	colRef := f.client.Collection(colPath)
	if colRef == nil {
		return "", errors.New("Invalid collection path. Must have even number of path tokens.")
	}

	i, err := strconv.Atoi(f.config["idSize"])
	if err != nil {
		return "", err
	}
	id, err := nanoid.Generate(f.config["idChars"], i)

	if err != nil {
		return "", err
	}
	return colPath + "/" + id, nil
}

// updateDoc pushes the data to the document at the given docPath. The changes are merged.
func (f fireFriend) updateDoc(docPath string, data map[string]any) error {

	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data, firestore.MergeAll)
	}

	return err
}

// setDoc pushes the data to the document at the given docPath. The document is overwritten.
func (f fireFriend) setDoc(docPath string, data map[string]any) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data)
	}

	return err
}

// deleteDoc removed the given document from the database.
func (f fireFriend) deleteDoc(docPath string) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Delete(f.ctx)
	}

	return err
}

func (f fireFriend) deleteField() any {
	return firestore.Delete
}

func (f fireFriend) refField(docPath string) any {
	ref, _ := f.docRef(docPath)
	return ref
}
