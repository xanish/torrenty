# Torrenty

A simple torrent client written in Go.

## TODO

- Sync peers from tracker every delay specified by tracker.
  - Need to do this because sometimes tcp connection breaks out.
  - Also noticed sometimes download does not finish due to missing piece.
- Gracefully stop and restart workers on refresh of peer data.

## Usage

TODO

## Features And Limitations

TODO

## BitTorrent Protocol

BitTorrent is a peer-to-peer (P2P) file-sharing protocol designed to distribute 
large amounts of data across a decentralized network of computers. Unlike 
traditional file transfer methods that rely on a central server, BitTorrent 
allows users to download files from multiple peers simultaneously, making it 
highly efficient for distributing large files.

## Components And Concepts

### Torrent File

- A file with .torrent extension containing metadata about the file(s) to be shared, such as file names, sizes, and folder structure.
- It includes the URL of the tracker, which helps the client find other peers sharing the file.
- It contains a hash (SHA-1) of each piece of the file, used to verify the integrity of the data.
- The file is encoded in bencode format (read as B-encode)

### Tracker

- A server that coordinates the file-sharing process providing peer discovery.
- Maintains a list of peers that have pieces of the file.

### Peers

- **Leechers:**
  - In process of downloading or have downloaded the file.
  - They don't share the pieces with other peers on the network.
  - Don't add any value to the network.
- **Seeders:**
  - Can have entire file or some pieces
  - Actively share those with others.
  - The goal is to have as many seeders as possible to ensure the file is available even if some peers leave the network.

### Pieces

- The file is divided into small fragments called pieces (e.g., 16KB).
- Each piece can be downloaded from any peer.
- Can download pieces randomly or rarest-first order.

### Swarm

- The group of peers sharing the same file.
- The swarm works together to keep the file available to all peers.

### Handshake

- Involves sending a message payload containing the bittorrent protocol, info hash and peer id

### Message

- Protocol messages are exchanged in the format: `<length prefix><message ID><payload>`.
- The length prefix is a 4B big-endian value.
- The message ID is a single decimal byte.
- The payload is message dependent.
- Message types include:
  - `keep-alive: <len=0000>`
  - `choke: <len=0001><id=0>`
  - `unchoke: <len=0001><id=1>`
  - `interested: <len=0001><id=2>`
  - `not interested: <len=0001><id=3>`
  - `have: <len=0005><id=4><piece index>`
  - `bitfield: <len=0001+X><id=5><bitfield>`
  - `request: <len=0013><id=6><index><begin><length>`
  - `piece: <len=0009+X><id=7><index><begin><block>`
  - `cancel: <len=0013><id=8><index><begin><length>`
  - `port: <len=0003><id=9><listen-port>`

### Bitfield

- A bitmaps which denotes which pieces of the entire file are available with the peer.
- This is the first message returned by peer once client connects to them via a handshake.

### Choking / Unchoking

- A mechanism to manage bandwidth and prioritize uploads to peers who are uploading to you.
- Peers "choke" others by temporarily stopping uploads to them if they are not reciprocating (tit-for-tat).

### Bencoding

- It supports four data types:
- **Integer:**
  - Starts with i, followed by the integer value, and ends with e.
  - Example: `i42e` => `42`
- **String:**
  - Starts with the length of the string as a number, followed by a colon, and then the string itself.
  - Example: `6:foobar` => `'foobar'`
- **List:**
  - Starts with l, followed by a list of bencoded values, and ends with e.
  - Example: `l4:spam3:eggi42ee` => `['spam', 'egg', 42]`
- **Dictionary:**
  - Starts with d, followed by a series of key-value pairs (where keys are strings and values are any bencoded type), and ends with e.
  - Keys must be sorted lexicographically.
  - Example: `d3:bar4:spam3:fooi42ee` => `{'bar': 'spam', 'foo': 42}`

#### Example File

```bencode
d
  6:announce18:udp://tracker.com:80
  4:info
    d
      12:piece lengthi131072e
      6:pieces20:xxxxxxxxxxxxxxxxxxxx...
      4:name4:file.txt
      5:lengthi1024e
    ee
e
```

**Breakdown of example:**
- `d`: Start of the top-level dictionary.
- `6:announce18:udp://tracker.com:80`: A key-value pair specifying the tracker URL.
- `4:info d`: Start of the info dictionary.
- `12:piece lengthi131072e`: The length of each piece in bytes.
- `6:pieces20:xxxxxxxxxxxxxxxxxxxx...`: The SHA-1 hashes of all pieces.
- `4:name4:file.txt`: The name of the file.
- `5:lengthi1024e`: The size of the file in bytes.
- `ee`: End of the info dictionary and the top-level dictionary.
- `e`: End of the top-level dictionary.

## Download Flow Using A Client

- **Load the Torrent File:**
  - Parse the bencoded .torrent file to read:
    - `announce`
    - `pieces`
    - `piece length`
    - `length`
    - `name`
- **Connect to the Tracker:**
  - An `HTTP.GET` request to the `announce` URL along with paramaters like:
    - `info_hash`
    - `peer_id`
    - `port` (One from 6881 to 6889)
    - `uploaded`
    - `downloaded`
    - `compact`
    - `left`
    - `numwant`
  - The tracker responds with:
    - `interval` which denotes how often you can reconnect to tracker to update it on what you have and don't, refresh peer list, etc.
    - `peers` containing list of IP addresses and port numbers, that are currently sharing the file.
- **Connect to Peers:**
  - Connect to the peers obtained using the BitTorrent protocol over TCP.
  - Exchanges handshake messages with each peer.
  - Verify the `info_hash` obtained in response with the `info_hash` of the file we want to download.
  - Disconnect if they do not match.
- **Exchange Bitfield:**
  - On successful handshake, peers send bitfield message, which are bitmaps indicating which pieces of the file they have.
- **Request Pieces:**
  - Select a piece to download.
  - Send `request` message to some peer which has the file available for download.
- **Download Pieces:**
  - Peer responds with the `piece` message which contains a block of the requested piece.
  - Verify the integrity of the piece once assembled.
  - Discard if the SHA1 hashes of the requested and downloaded piece do not match and retry.
- **Upload Pieces:**
  - Can share the available pieces to other peers.
  - The more pieces we have and share, the better the chances of getting more pieces from other peers.
- **Complete Download and Assemble File:**
  - Ensure all pieces are downloaded and correctly assembled into the final file.
  - Continue seeding.
- **Update Tracker:**
  - Periodically, update the tracker (based on received interval).
  - Send information like, the number of pieces downloaded, uploaded, etc.
  - Can request more peers for downloading.
- **Handle Disconnects and Errors:**
  - Gracefully handle peer disconnections and timeouts
  - Retry connections as necessary to resume download.

## References

- https://wiki.theory.org/BitTorrentSpecification