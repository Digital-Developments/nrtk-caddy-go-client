# Newsroom Toolkit Go Client 
Lightweight website server instance powered by [Newsroom Toolkit](https://nrtk.app). The client of the [Newsroom Toolkit Hub](https://github.com/Digital-Developments/nrtk-hub) and a derivative work of the [Newsroom Toolkit Python Client](https://github.com/Digital-Developments/nrtk-client-python).

Newroom Toolkit provides a modern CMS and a powerful API focused on your needs. This open source package adds a layer of independence, resiliency, and security to your publication hosted on Newsroom Toolkit. Here is a [blog](https://joeface.com) is powered by this library.

## Key Features
* All-in-one solution for a self-hosted websites and mirroring
* Automated content synchronization
* Local backups and version control (you always have a snapshot of your content)
* SEO optimization via sitemap.xml
* Basic error page templates
* Open Source

## Run
This section assumes that you already have Go [installed](https://go.dev/doc/install) on your machine: 
```
$ go mod download && go run .
```

## Docker
To build it with Docker simply run from the project dir:
```
$ docker build -t nrtk-go-client .
```

## License
The project is licensed under the GNU General Public License v3.0 (see the [LICENSE](LICENSE) file).