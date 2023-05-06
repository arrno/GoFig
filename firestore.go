package main

import (
	"context"
	"errors"
	"strconv"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/aidarkhanov/nanoid"
	"google.golang.org/api/option"
)

// TODO implement this
type Firestore interface {
	GetDocData(docPath string) (map[string]any, error)
	GenDocPath(colPath string) (string, error)
	UpdateDoc(docPath string, data map[string]any) error
	SetDoc(docPath string, data map[string]any) error
	DeleteDoc(docPath string) error
}

type Firefriend struct {
	client *firestore.Client
	ctx    context.Context
	config map[string]string
}

func NewFirestore(keyPath string) (*Firefriend, func(), error) {

	ctx := context.Background()

	sa := option.WithCredentialsFile(keyPath)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, func() {}, err
	}

	client, err := app.Firestore(ctx)

	config := map[string]string{
		"idChars": "",
		"idSize":  "30",
	}

	f := Firefriend{
		client,
		ctx,
		config,
	}
	return &f, func() { client.Close() }, err

}

func (f Firefriend) docRef(path string) (*firestore.DocumentRef, error) {

	ref := f.client.Doc(path)

	if ref == nil {
		return ref, errors.New("Unable to create doc reference.")
	}

	return ref, nil

}

func (f Firefriend) doc(path string) (*firestore.DocumentSnapshot, error) {

	ref, err := f.docRef(path)

	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(f.ctx)

	return snap, err
}

func (f Firefriend) GetDocData(docPath string) (map[string]any, error) {

	snap, err := f.doc(docPath)

	if err != nil {
		return map[string]interface{}{}, err

	}
	return snap.Data(), nil
}

func (f Firefriend) GenDocPath(colPath string) (string, error) {

	colRef := f.client.Collection(colPath)
	if colRef == nil {
		return "", errors.New("Invalid collection path. Must have even number of path tokens.")
	}

	i, err := strconv.Atoi(f.config["Idsize"])
	if err != nil {
		return "", err
	}
	id, err := nanoid.Generate(f.config["Idchars"], i)

	if err != nil {
		return "", err
	}
	return colPath + "/" + id, nil
}

func (f Firefriend) UpdateDoc(docPath string, data map[string]any) error {

	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data, firestore.MergeAll)
	}

	return err
}

func (f Firefriend) SetDoc(docPath string, data map[string]any) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Set(f.ctx, data)
	}

	return err
}

func (f Firefriend) DeleteDoc(docPath string) error {
	ref, err := f.docRef(docPath)

	if err == nil {
		_, err = ref.Delete(f.ctx)
	}

	return err
}
