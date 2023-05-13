package main

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

// Firestore is an interface that expresses what a NoSQL database dependency should do.
type Firestore interface {
	GetDocData(docPath string) (map[string]any, error)
	GenDocPath(colPath string) (string, error)
	UpdateDoc(docPath string, data map[string]any) error
	SetDoc(docPath string, data map[string]any) error
	DeleteDoc(docPath string) error
	DeleteField() any
	RefField(docPath string) any
	Name() string
}

// Firefriend is our implementation/wrapper for google's firestore client.
type Firefriend struct {
	client *firestore.Client
	ctx    context.Context
	config map[string]string
}

// NewFirestore is a Firefriend factory
func NewFirestore(keyPath string) (*Firefriend, func(), error) {

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

	f := Firefriend{
		client,
		ctx,
		config,
	}
	return &f, func() { client.Close() }, err

}

// Name returns a hash for the underlying firestore database name. This may
// be useful for guarding against pushing changes to the wrong database.
func (f Firefriend) Name() string {
	s := strings.Split(f.client.Doc("__init__/__init__").Path, "__init__/__init__")[0]
	return strings.Split(s, "/databases/(default)/documents/")[0]
}

// docRef converts a string path to a firestore document reference.
func (f Firefriend) docRef(path string) (*firestore.DocumentRef, error) {

	ref := f.client.Doc(path)

	if ref == nil {
		return ref, errors.New("Unable to create doc reference.")
	}

	return ref, nil

}

// doc converts a string path to a firestore document snapshot.
func (f Firefriend) doc(path string) (*firestore.DocumentSnapshot, error) {

	ref, err := f.docRef(path)

	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(f.ctx)

	return snap, err
}

// getDocData attempts to read the specified document. If the document exists, it returns the underlying data.
func (f Firefriend) GetDocData(docPath string) (map[string]any, error) {

	snap, err := f.doc(docPath)
	if !snap.Exists() {
		return map[string]any{}, nil
	}

	if err != nil {
		return map[string]interface{}{}, err

	}
	return snap.Data(), nil
}

// GenDocPath is used to generate new unique document path given a collection path.
func (f Firefriend) GenDocPath(colPath string) (string, error) {

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

// UpdateDoc pushes the data to the document at the given docPath. The changes are merged.
func (f Firefriend) UpdateDoc(docPath string, data map[string]any) error {

	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data, firestore.MergeAll)
	}

	return err
}

// SetDoc pushes the data to the document at the given docPath. The document is overwritten.
func (f Firefriend) SetDoc(docPath string, data map[string]any) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data)
	}

	return err
}

// DeleteDoc removed the given document from the database.
func (f Firefriend) DeleteDoc(docPath string) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Delete(f.ctx)
	}

	return err
}

// DeleteField returns the firestore Delete value which can be set on a nested
// data field within a Set/Update operation. The field will be removed when
// UpdateDoc or SetDoc is called.
func (f Firefriend) DeleteField() any {
	return firestore.Delete
}

// RefField is guaranteed to return something that will be properly
// serialized/deserialized and stored as a firestore document reference
func (f Firefriend) RefField(docPath string) any {
	ref, _ := f.docRef(docPath)
	return ref
}
