effio
=====

[COMING SOON](http://tobert.github.io/post/2014-04-29-a-quick-prototype.html)

Tools for crunching data from fio, the Flexible IO Tester

Usage
-----

Note: this is an exploration CLI design! Not even coded yet ...

Wherever possible, effio sets sane defaults. For example, if you want a line graph
from a single JSON file. This will automatically graph 'lat' for all jobs in the file.
There will be one line for each of the read/write/trim/mixed datasets that contain a 'lat'
entry.

    effio line_graph -json trivial.json

You can also be specific. This will generate one line for the job trivial-readwrite-1g
jobs' read IO completion latency.

    effio histogram -json trivial.json \
      -jobname trivial-readwrite-1g \
      -iotype read -metric clat

Some things cannot be guessed when using the latency logs. There's nothing there but a bunch of
numbers so you need to provide metadata on the command line.

This example also shows the -buckets option that is specific to the histogram renderer.

    effio histogram -csv 0x5000c5000d7f96d9_lat.log \
      -title "7200 RPM SAS drive read latency" \
      -xlabel "time" -ylabel "latency" -legend topright \
      -buckets 20 \
      -map 0x5000c5000d7f96d9="Seagate 7200RPM SAS"

One common task -- and one I don't look forward to -- is comparing data from multiple files, for
both CSV and JSON. When multiple files are specified on the command line, the order IS PRESERVED.
The rest - choosing which data to graph - is the same as above. Leaving the jobname blank means
select from all job names and so on.

    effio scatter -json sda.json -json sdb.json 
