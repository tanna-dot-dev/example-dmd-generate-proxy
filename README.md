# Example Dependency Management Data Caching Proxy

An example implementation of a caching proxy for HTTP traffic for the [dependency-management-data](https://dmd.tanna.dev) project.

> [!NOTE]
> This is [not yet](https://gitlab.com/tanna.dev/dependency-management-data/-/merge_requests/263) fully supported in dependency-management-data.

> [!IMPORTANT]
> This is very much a work in progress, hacky, and not-quite-ready-for-real-usage solution, worked on during [Encore's Launch Week](https://encore.dev/launchweek).
>
> This will be cleaned up over the next couple of weeks to make it a production worthy solution, including extracting out authentication to allow for easier management of clients.

## Running locally

```bash
encore run
```

While `encore run` is running, open [http://localhost:9400/](http://localhost:9400/) to view Encore's [local developer dashboard](https://encore.dev/docs/observability/dev-dash).

## Using the API

If, for instance, you wanted to perform a pURL-based lookup in Ecosystems for:

```
https://packages.ecosyste.ms/api/v1/packages/lookup?purl=pkg:golang/dmd.tanna.dev
```

Instead, you could proxy it through this service like so:

```sh
curl 'http://localhost:4000/packages.ecosyste.ms/api/v1/packages/lookup?purl=pkg:golang/dmd.tanna.dev' -i -H 'Authorization: Bearer me'
```

This then caches the request in the PostgreSQL database.

To clear the cache (and refetch the data asynchronously, you can perform a `DELETE` request to a given URL:

```sh
curl -X DELETE 'http://localhost:4000/packages.ecosyste.ms/api/v1/packages/lookup?purl=pkg:golang/dmd.tanna.dev' -i -H 'Authorization: Bearer me'
```

## License

AGPL-3.0
