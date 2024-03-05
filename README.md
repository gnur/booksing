# booksing

<img src="./gopher.png" width="350" alt="nerdy gopher">
A tool to browse epubs.

Kind of inspired by https://github.com/geek1011/BookBrowser/

## Installation

Download an appropriate release from the [release](https://github.com/gnur/booksing/releases) page

## DISCLAIMER

**_Please, please, please be careful. I'm pretty sure it shouldn't eat up your collection, but it might. If your are testing booksing, please copy your files into the import dir. Do not move them until you feel comfortable with booksing._**

## Features

- Easy-to-use
- List view
- Automatic deletion of duplicates and unparsable epubs
- Automatic sorting of books based on Author
- Users meilisearch for blazing fast fuzzy search

## Configuration

Set the following env vars to configure booksing:

| env var               | default                 | required | purpose                                                                                                             |
| --------------------- | ----------------------- | -------- | ------------------------------------------------------------------------------------------------------------------- |
| BOOKSING_BINDADDRESS  | `localhost:7132`        | :x:      | The bind address, if external access is needed this should be changed to `:7132`                                    |
| BOOKSING_BOOKDIR      | `./books/`              | :x:      | The directory where books are stored after importing                                                                |
| BOOKSING_FAILDIR      | `./failed`              | :x:      | The directory where books are moved if the import fails                                                             |
| BOOKSING_IMPORTDIR    | `./import`              | :x:      | The directory where booksing will periodically look for books                                                       |
| BOOKSING_LOGLEVEL     | `info`                  | :x:      | determines the loglevel, supported values: error, warning, info, debug                                              |
| BOOKSING_MAXSIZE      | `0`                     | :x:      | If set, any epub larger than this size in bytes will be automatically deleted, can be useful with limited diskspace |
| BOOKSING_TIMEZONE     | `Europe/Amsterdam`      | :x:      | Timezone used for storing all time information                                                                      |
| BOOKSING_MEILIADDRESS | `http://localhost:7700` | :x:      | Address to find meilisearch                                                                                         |
| BOOKSING_MEILISECRET  | `""`                    | :x:      | Secret to connect to meilisearch                                                                                    |

## Example first run

```

$ mkdir booksing booksing/failed booksing/import booksing/db
$ cd booksing
$ wget 'https://github.com/gnur/booksing/releases/download/v9.0.5/booksing_9.0.5_linux_x86_64.tar.gz'
$ tar xzf booksing*
$ mkdir books db failed import
$ ./booksing &
$ mv ~/library/*.epub import/
# visit localhost:7132 to see the books in the interface
```

## systemd unit file

There is an example systemd unit file available on the releases page, can also be found in `includes/booksing.service`
