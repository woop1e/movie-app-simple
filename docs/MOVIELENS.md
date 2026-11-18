# MovieLens development data

The local demo catalog can be expanded with GroupLens MovieLens latest-small. The importer matches titles already in the Yozigo catalog so their existing poster and description remain intact. MovieLens does not include poster art, runtime, synopsis, language, or content certification; imported records show a generated poster tile, "Runtime unavailable", "Unknown", and "NR" for those fields.

Download and import from the backend directory:

```sh
mkdir -p ../work/datasets/movielens
curl -L https://files.grouplens.org/datasets/movielens/ml-latest-small.zip -o ../work/datasets/movielens/ml-latest-small.zip
unzip -o ../work/datasets/movielens/ml-latest-small.zip -d ../work/datasets/movielens
go run ./cmd/import-movielens
```

The importer reads CSV files from `../work/datasets/movielens/ml-latest-small` by default. Set `MOVIELENS_DIR`, `MONGODB_URI`, and `MONGODB_DATABASE` to use another location or database. It can be run again without adding duplicate titles, profiles, or rating events.

MovieLens users are anonymized demonstration profiles with `.invalid` email addresses and no password credentials; they cannot sign in. Their historical ratings are used only to provide a collaborative-filtering neighborhood for new app users. Read the dataset's included README and GroupLens terms before using the data outside this educational local demo.
