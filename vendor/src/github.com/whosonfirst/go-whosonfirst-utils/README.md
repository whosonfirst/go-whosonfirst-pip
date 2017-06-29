# go-whosonfirst-utils

Go tools and utilities for working with Who's On First documents.

## Install

You will need to have both `Go` and the `make` programs installed on your computer. Assuming you do just type:

```
make bin
```

All of this package's dependencies are bundled with the code in the `vendor` directory.

## Tools

### wof-cat

Expand and concatenate one or more Who's On First IDs and print them to `STDOUT`.

```
./bin/wof-cat -h
Usage of ./bin/wof-cat:
  -alternate
    	Encode URI as an alternate geometry
  -extras string
    	A comma-separated list of extra information to include with an alternate geometry (optional)
  -function string
    	The function of the alternate geometry (optional)
  -root string
    	If empty defaults to the current working directory + "/data".
  -source string
    	The source of the alternate geometry
  -strict
    	Ensure that the source for an alternate geometry is valid (see also: go-whosonfirst-sources)
```

For example, assuming you are in the `whosonfirst-data` repo, to dump the record for [San Francisco](https://whosonfirst.mapzen.com/spelunker/id/85922583/) (or `data/859/225/83/85922583.geojson`) you would type:

```
$> wof-cat 85922583 | less
{
  "id": 85922583,
  "type": "Feature",
  "properties": {
    "edtf:cessation":"uuuu",
    "edtf:inception":"uuuu",
    "geom:area":0.061408,
    "geom:area_square_m":600307527.980658,
    "geom:bbox":"-123.173825,37.63983,-122.28178,37.929824",
    "geom:latitude":37.759715,
    "geom:longitude":-122.693976,
    "gn:elevation":16,
    "gn:latitude":37.77493,
    "gn:longitude":-122.41942,
    "gn:population":805235,
    "iso:country":"US",
    "lbl:bbox":"-122.51489,37.70808,-122.35698,37.83239",
    "lbl:latitude":37.778008,
    "lbl:longitude":-122.431272,
    "mps:latitude":37.778008,
    "mps:longitude":-122.431272,
    "mz:hierarchy_label":1,
    "name:chi_x_preferred":[
        "\u65e7\u91d1\u5c71"
    ],
    "name:chi_x_variant":[
        "\u820a\u91d1\u5c71"
    ],
    "name:eng_x_colloquial":[
        "City by the Bay",
        "City of the Golden Gate",
        "Fog City",

...and so on
```

### wof-compare

```
./bin/wof-compare -h
Usage of ./bin/wof-compare:
  -filelist
    	Read WOF IDs from a "file list" document.
```

Compare one or more Who's On First documents against multiple sources, reading IDs from the command line or a "file list" document. Current sources are:

| Source | URI | Notes |
| :--- | :--- | :--- |
| `wof` | https://whosonfirst.mapzen.com/data/ | |
| `github` | https://raw.githubusercontent.com/:REPO:/whosonfirst-data/master/data/ | the `:REPO:` bit is discussed below |
| `s3` | https://s3.amazonaws.com/whosonfirst.mapzen.com/data/ | |

_It is not possible to compare local files on disk. Yet._

For example:

```
./bin/wof-compare 890537261 85898837
wofid,match,github,wof,s3
890537261,MATCH,2e4d929a9faad66043f3c5790363ba3e,2e4d929a9faad66043f3c5790363ba3e,2e4d929a9faad66043f3c5790363ba3e
85898837,MATCH,b64b923c27d68abfddd234ee6c8c584a,b64b923c27d68abfddd234ee6c8c584a,b64b923c27d68abfddd234ee6c8c584a
```

You can also pass a "file list" document which is pretty much what it sounds like: A list of WOF documents, one per line. For example here is how you might generate a file list for all the descendants of Helsinki using the [wof-api](https://github.com/whosonfirst/go-whosonfirst-api#wof-api) tool:

```
./bin/wof-api -param method=whosonfirst.places.getDescendants -param id=101748417 -param api_key=mapzen-xxxxxx -filelist -filelist-prefix /usr/local/data/:REPO:/data -paginated 
/usr/local/data/whosonfirst-data/data/858/988/21/85898821.geojson
/usr/local/data/whosonfirst-data/data/858/988/25/85898825.geojson
/usr/local/data/whosonfirst-data/data/858/988/37/85898837.geojson
/usr/local/data/whosonfirst-data/data/858/988/43/85898843.geojson
/usr/local/data/whosonfirst-data/data/858/988/45/85898845.geojson
/usr/local/data/whosonfirst-data/data/858/988/47/85898847.geojson
/usr/local/data/whosonfirst-data/data/858/988/51/85898851.geojson
/usr/local/data/whosonfirst-data/data/858/988/55/85898855.geojson
/usr/local/data/whosonfirst-data/data/858/988/61/85898861.geojson
/usr/local/data/whosonfirst-data/data/858/988/65/85898865.geojson
/usr/local/data/whosonfirst-data/data/858/988/67/85898867.geojson
/usr/local/data/whosonfirst-data/data/858/988/75/85898875.geojson
/usr/local/data/whosonfirst-data/data/858/988/81/85898881.geojson
/usr/local/data/whosonfirst-data/data/858/988/85/85898885.geojson
... and so on
```

Now, let's imagine you wrote those results to a file called `helsinki.txt`. You can compare all the WOF IDs listed like this:

```
./bin/wof-compare -filelist helsinki.txt 
wofid,match,github,s3,wof
85898821,MATCH,9b0b76f328a64d9f0eb3ab604ea855eb,9b0b76f328a64d9f0eb3ab604ea855eb,9b0b76f328a64d9f0eb3ab604ea855eb
85898825,MATCH,a799cc0f2d46c3b0c94c2f014286b16d,a799cc0f2d46c3b0c94c2f014286b16d,a799cc0f2d46c3b0c94c2f014286b16d
85898837,MATCH,b64b923c27d68abfddd234ee6c8c584a,b64b923c27d68abfddd234ee6c8c584a,b64b923c27d68abfddd234ee6c8c584a
85898843,MATCH,c557962d544f66c51777629568dcb542,c557962d544f66c51777629568dcb542,c557962d544f66c51777629568dcb542
85898845,MATCH,6e7e8e5ab00b8e92bc3f9c1de512c866,6e7e8e5ab00b8e92bc3f9c1de512c866,6e7e8e5ab00b8e92bc3f9c1de512c866
85898847,MATCH,0822020aaed4b53379df793cbaebe2b5,0822020aaed4b53379df793cbaebe2b5,0822020aaed4b53379df793cbaebe2b5
85898851,MATCH,9dc34d1ef12f38e329b5e8b3f8bd8533,9dc34d1ef12f38e329b5e8b3f8bd8533,9dc34d1ef12f38e329b5e8b3f8bd8533
85898855,MATCH,8f099ef74842b1ec42990d7a4dc165a8,8f099ef74842b1ec42990d7a4dc165a8,8f099ef74842b1ec42990d7a4dc165a8
85898861,MATCH,38f0fb958a9fd997b22046923e9fa443,38f0fb958a9fd997b22046923e9fa443,38f0fb958a9fd997b22046923e9fa443
85898865,MATCH,989d91f616a4a71fcc6c4b4da0850d4c,989d91f616a4a71fcc6c4b4da0850d4c,989d91f616a4a71fcc6c4b4da0850d4c
85898867,MATCH,94ad036fcf2bd7b1c0f8bda661e50e2f,94ad036fcf2bd7b1c0f8bda661e50e2f,94ad036fcf2bd7b1c0f8bda661e50e2f
85898875,MATCH,ec3114af9ef10bf18c9e01a016db31d8,ec3114af9ef10bf18c9e01a016db31d8,ec3114af9ef10bf18c9e01a016db31d8
85898881,MATCH,c6f576fee8e47d06eb6b9baae2dffe89,c6f576fee8e47d06eb6b9baae2dffe89,c6f576fee8e47d06eb6b9baae2dffe89
85898885,MATCH,5c16f782f59366e52b8eb3f07dbdac10,5c16f782f59366e52b8eb3f07dbdac10,5c16f782f59366e52b8eb3f07dbdac10
85898887,MATCH,80a8bb00344e65f69bc5e0891e45f491,80a8bb00344e65f69bc5e0891e45f491,80a8bb00344e65f69bc5e0891e45f491
85898895,MATCH,7f4d38818ed11cb5f484557a6e8b723c,7f4d38818ed11cb5f484557a6e8b723c,7f4d38818ed11cb5f484557a6e8b723c
85898897,MATCH,706b9266f7815f79834b410189a7e917,706b9266f7815f79834b410189a7e917,706b9266f7815f79834b410189a7e917
... and so on
```

#### Important

Do you see the way we're pasing "/usr/local/data/:REPO:/data" to the `-filelist-prefix` flag in the example above? That's important. The [wof-api](https://github.com/whosonfirst/go-whosonfirst-api#wof-api) tool's "file list" writer will replace the string `:REPO:` with the actual `wof:repo` returned by the API when it's generating the file list.

Similarly, the default URI for the `github` source is `https://raw.githubusercontent.com/:REPO:/whosonfirst-data/master/data/`. When processing file list documents the code will try to determine the repo name from a path and replacing `:REPO:` accordingly. If no repo can be determined then `whosonfirst-data` will be assumed.

### wof-d2fc

Create a GeoJSON `FeatureCollection` from a Git diff (so "diff2featurecollection" or "d2fc"). What that really means is produce FeatureCollection from any list of files formatted per the output of `git diff --name-only`. Basically I wanted a way to easily snapshot one or records in advance of a Git reset so that's what this tool is optimized for.

For example:

```
$> git diff --name-only HEAD..01a6fdd25b7de2d3da7aa2f53f4f44a7efe81c47 | /usr/local/bin/wof-d2fc -repo /usr/local/data/whosonfirst-data-venue-us-ca | python -mjson.tool
{
    "features": [
        {
            "bbox": [
                -122.46228,
                37.783038,
                -122.46228,
                37.783038
            ],

... and so on
```

It is left to users to filter out any non-GeoJSON files from the list passed to `wof-d2fc`.

### wof-geojsonls-dump

Dump one or more directories containing Who's On First documents as line-separated (encoded) GeoJSON.

```
./bin/wof-geojsonls-dump -h
Usage of ./bin/wof-geojsonls-dump:
  -exclude-deprecated
    	Exclude records that have been deprecated.
  -exclude-superseded
    	Exclude records that have been superseded.
  -out string
    	Where to write records (default is STDOUT)
  -processes int
    	The number of concurrent processes to use (default 16)
  -timings
    	Print timings
```

For example:

```
./bin/wof-geojsonls-dump --exclude-deprecated --exclude-superseded /usr/local/data/whosonfirst-data-venue-* > /tmp/venues-all.txt
```

### wof-geojsonls-validate

Ensure that all the records in a GeoJSON LS dump are valid JSON.

```
./bin/wof-geojsonls-validate -h
Usage of ./bin/wof-geojsonls-validate:
  -processes int
    	The number of concurrent processes to use (default 16)
  -stats
    	Be chatty, with counts and stuff
  -strict
    	Whether or not to trigger a fatal error when invalid JSON is encountered
```

For example:

```
./bin/wof-geojsonls-validate -processes 128 -stats -strict /usr/local/data-ext/venues/venues-20170628.txt
2017/06/29 16:31:37 /usr/local/data-ext/venues/venues-20170628.txt 21650210 records processed in 17m14.809141146s
```

_Note that we are only checking that each line can be successfully parsed as JSON and not validating any GeoJSON related specifics._

### wof-ensure-property

Crawl a WOF repo reporting any files that are missing a given property.

```
./bin/wof-ensure-property -h
Usage of ./bin/wof-ensure-property:
  -processes int
    	The number of concurrent processes to use (default 16)
  -property string
    	The dotted notation for the property whose existence you want to test.
  -repo string
    	The WOF repo whose files you want to test. (default "/usr/local/data/whosonfirst-data")
```

For example:

```
./bin/wof-ensure-property -processes 64 -property wof:parent_id -repo /usr/local/data/whosonfirst-data
id,path,details
1108810255,/usr/local/data/whosonfirst-data/data/110/881/025/5/1108810255.geojson,missing 'properties.wof:parent_id'
1108803081,/usr/local/data/whosonfirst-data/data/110/880/308/1/1108803081.geojson,missing 'properties.wof:parent_id'
1108803083,/usr/local/data/whosonfirst-data/data/110/880/308/3/1108803083.geojson,missing 'properties.wof:parent_id'
1108803089,/usr/local/data/whosonfirst-data/data/110/880/308/9/1108803089.geojson,missing 'properties.wof:parent_id'
1108803101,/usr/local/data/whosonfirst-data/data/110/880/310/1/1108803101.geojson,missing 'properties.wof:parent_id'
1108803107,/usr/local/data/whosonfirst-data/data/110/880/310/7/1108803107.geojson,missing 'properties.wof:parent_id'

... and so on
```

#### Caveats

As of this writing `wof-ensure-property` will:

* skip "alt" files
* only check for the existence of a property; it will not evaluate its value

### wof-expand

Expand one or more Who's On First IDs to their absolute paths and print them to `STDOUT`.

```
./bin/wof-expand -h
Usage of ./bin/wof-expand:
  -alternate
    	Encode URI as an alternate geometry
  -extras string
    	A comma-separated list of extra information to include with an alternate geometry (optional)
  -function string
    	The function of the alternate geometry (optional)
  -prefix string
    	Prepend this prefix to all paths
  -root string
    	The directory where Who's On First records are stored. If empty defaults to the current working directory + "/data".
  -source string
    	The source of the alternate geometry
  -strict
    	Ensure that the source for an alternate geometry is valid (see also: go-whosonfirst-sources)
```

### wof-hash

_Please write me_

## See also

* https://github.com/whosonfirst/go-whosonfirst-uri
