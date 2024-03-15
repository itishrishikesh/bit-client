This is a simple bit torrent client written in go, this project is written purely for educational purposes and is inspired from this blog post - https://blog.jse.li/posts/torrent/.

This client only works with HTTP/TCP torrents, and only works with `.torrent` files.

The usage is quite simple,

```bash
./bit-client in=/path/to/any.torrent_file out=/path/to/downloaded/some.file
```