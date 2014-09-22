/*
 * Copyright 2014 Albert P. Tobey <atobey@datastax.com> @AlTobey
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

var APP = {};

// some functions used in selectors for extracting fields from the summaries
APP.fields = {
  average:     function (smry) { return smry.average; },
  median:      function (smry) { return smry.median;  },
  min:         function (smry) { return smry.min;     },
  max:         function (smry) { return smry.max;     },
  count:       function (smry) { return smry.count;   },
  stdev:       function (smry) { return smry.stdev;   },
  percentiles: function (smry) {
    // effio's summarize adds 102 percentile values to each summary
    // that can be used to create high-resolution graphs
    return d3.keys(smry.percentiles)
      // sort by timestamp instead of value, will be unevenly distributed but whatevs
      .sort(function (a,b) {
        if (a.hasOwnProperty("idx")) {
          return smry.percentiles[a].idx - smry.percentiles[b].idx;
        } else {
          return smry.percentiles[a].time - smry.percentiles[b].time;
        }
      })
      // get the value at percentiles[d]
      .map(function (key) { return +smry.percentiles[key].value; });
  }
};

// builds the bootstrap layout then puts all the controls into the containers
APP.render_page_layout = function (target) {
  APP.setup_chart_controls(target);
  APP.build_nav();
};

// do a first level of filtering then call into C3 or d3.box
// { benchmark: "foobar", sample: "all", type: "c3.line", log_type: "lat", fun: APP.fields.average }
APP.chart = function (target, devices, chart1, chart2) {
  //d3.select("#top_mid").text(benchmark + " / " + sample_type);
  //
  console.log("APP.chart(", target, devices, chart1, {}, ");");

  var chart1_data = APP.filter_summaries(devices, chart1.benchmark, chart1.log_type, chart1.rotational);
  console.log("Chart 1 selected summaries", chart1_data);

  var ctype = chart1.type.split("."); // c3.line, c3.bar, d3.box
  if (ctype[0] === "c3") {
    var chart = APP.c3chart(target, chart1_data, ctype[1], chart1);

    // disabled for now (2014-09-10)
    if (chart2.hasOwnProperty("log_type") && chart2["log_type"] != "off") {
      var chart2_data = APP.filter_summaries(devices, chart2.benchmark, chart2.log_type, chart2.rotational);
      chart.load({ columns: APP.summaries_to_c3(chart2_data, chart2.sample, chart2.fun), type: "line", style: "dashed" });
    }
  // doesn't make sense to do two dimensions on a box chart (for now)
  } else if (ctype[0] === "d3" && ctype[1] === "box") {
    APP.d3box(target, chart1_data, chart1.sample, chart1.fun);
  } else {
    alert("Invalid chart type: '" + chart1.type + "'");
  }
};

APP.filter_summaries = function (devices, benchmark, log_type, rotational) {
  // finds the summaries that contain the benchmark requested
  return APP.summaries
    // sort by device name to keep layout consistent
    .sort(function (a,b) {
      if (a.fio_command.device.name > b.fio_command.device.name) { return  1; }
      if (a.fio_command.device.name < b.fio_command.device.name) { return -1; }
      return 0;
    })
    // only display selected devices
    .filter(function (d) { return devices.hasOwnProperty(d.fio_command.device.name); })
    // filter by log type (bw, iops, lat)
    .filter(function (d) { return d.log_type === log_type; })
    // only display the selected benchmark name
    .filter(function (d) { return d.fio_command.fio_name === benchmark; })
    // quick split on ssd/hdd
    .filter(function (d) {
      if (rotational === "All") {
        return true;
      } else {
        return d.fio_command.device.rotational === (rotational === "HDD");
      }
    });
};

// format the data for C3
APP.summaries_to_c3 = function (data, sample_type, fun) {
  return data.map(function (summary) {
    var col = [];
    if (sample_type === "percentiles_bin") {
      var pc = summary["percentiles"];
        col = d3.keys(pc)
          .sort(function (a,b) { return pc[a].value - pc[b].value; })
          .map(function (key) { return +pc[key].value; })
          .reduce(function (a,b) { return a.concat(b) });
    } else {
      col = summary[sample_type].map(fun);
    }

    // C3 expects the name as the first element, then the column data
    col.unshift(summary.fio_command.device.name);

    return col;
  });
};

// draw charts with c3.js
APP.c3chart = function (target, data, chart_type, config) {
  console.log("APP.c3chart", data, chart_type, config);

  var cols = APP.summaries_to_c3(data, config.sample, config.fun);

  var ytxt = "Latency (microseconds)"
  if (config.log_type === "bw") {
    ytxt = "Bandwidth"
  } else if (config.log_type === "iops") {
    ytxt = "IOPS"
  }

  return c3.generate({
    bindto: target,
    data: { columns: cols, type: chart_type, colors: APP.device_colors },
    axis: {
      y: { label: { text: ytxt, position: "outer-middle" } },
      x: { label: { text: "Time Offset (0-10 minutes)" } }
    }
  });
};

// use the percentiles to get d3 box.js to work
// only supports 4-5 elements!
APP.d3box = function (target, summaries, sample_type, fun) {
  console.log("APP.d3box", summaries, sample_type);
  d3.select(target).selectAll("svg").remove();

  var margin = {top: 10, right: 40, bottom: 20, left: 40};
  var width = 100 - margin.left - margin.right;
  var height = 600 - margin.top - margin.bottom;
  var max = -Infinity;
  var min = Infinity;

  var data = summaries.map(function (smry, i) {
    return d3.keys(smry.percentiles)
      .filter(function (key) { return +key < 99.1; })
      .sort(function (a,b) { return a - b; })
      .map(function (key,i) {
        var val = smry.percentiles[key].value;

        // side-effects to save a pass over the data
        if (val > max) { max = val; }
        if (val < min) { min = val; }

        return val;
      });
  });

  var devices = summaries.map(function (smry, i) {
    return smry.fio_command.device.name;
  });

  var chart = d3.box()
    .whiskers(APP.iqr(1.5))
    .width(width)
    .height(height)
    .domain([min, max]);

  var svg = d3.select("#mid_middle").selectAll("svg")
    .data(data)
    .enter().append("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.bottom + margin.top);

  var bg = svg.append("g")
    .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

  bg.append("rect")
    .attr("class", "box-whisker-bg")
    .attr("width", "100%")
    .attr("height", "100%");

  bg.append("text")
      .text(function (d,i) { return devices[i]; })
      .attr("class", "box-device-name")
      .attr("transform", "rotate(90)");

  var box = bg.append("g")
    .attr("class", "box")
    .attr("transform", "translate(" + margin.left + "," + margin.top + ")")
    .call(chart);
};

// needed by d3.box to compute inter-quartile range
APP.iqr = function (k) {
  return function(d, i) {
    var q1 = d.quartiles[0],
    q3 = d.quartiles[2],
    iqr = (q3 - q1) * k,
    i = -1,
    j = d.length;
    while (d[++i] < q1 - iqr);
    while (d[--j] > q3 + iqr);
    return [i, j];
  };
};

// set up 9 regions on the screen using bootstrap
// see also: ../css/app.css
APP.setup_chart_controls = function (target) {
  var body = d3.select(target);
  body.selectAll(".container-fluid").remove();
  var ctr = body.append("div").classed({"container-fluid": true});

  var top_div = ctr.append("div").classed({"row": true}).attr("id", "top_row");
  var mid_div = ctr.append("div").classed({"row": true}).attr("id", "mid_row");
  var bot_div = ctr.append("div").classed({"row": true}).attr("id", "bot_row");

  top_div.append("div").classed({"col-md-2": true}).attr("id", "top_left");
  top_div.append("div").classed({"col-md-8": true}).attr("id", "top_middle");
  top_div.append("div").classed({"col-md-2": true}).attr("id", "top_right");
  mid_div.append("div").classed({"col-md-2": true}).attr("id", "mid_left");
  mid_div.append("div").classed({"col-md-8": true}).attr("id", "mid_middle");
  mid_div.append("div").classed({"col-md-2": true}).attr("id", "mid_right");
  bot_div.append("div").classed({"col-md-2": true}).attr("id", "bot_left");
  bot_div.append("div").classed({"col-md-8": true}).attr("id", "bot_middle");
  bot_div.append("div").classed({"col-md-2": true}).attr("id", "bot_right");
};

// called whenver a change is made in the controls on the left
// accesses the form elements to get values then rerenders
APP.change = function (side) {
  var devs = {};
  var chart1 = {name: "chart1"};
  var chart2 = {name: "chart2"};

  // both charts must always show the same devices mostly because there isn't a good place
  // to put the device selection for now
  d3.selectAll(".device-checkbox input")
    .each(function (d) { if (this.checked == true) { devs[this.value] = true; } });

  // left chart / right chart
  [chart1, chart2].forEach(function (chart) {
    // adds the chart name to the id so this code isn't repeated for left/right side
    var id = function (suffix) {
      console.log("APP.change.id returns: ." + chart.name + "-" + suffix);
      return "." + chart.name + "-" + suffix;
    };

    d3.selectAll(id("benchmark-radio input"))
      .each(function (d) { if (this.checked == true) { chart.benchmark = this.value; } });

    d3.selectAll(".ddir-radio input")
      .each(function (d) { if (this.checked == true) { chart.ddir = this.value; } });

    d3.selectAll(".pcntl-radio input")
      .each(function (d) { if (this.checked == true) { chart.pcntl = this.value; } });

    d3.selectAll(id("chart-type-radio input"))
      .each(function (d) { if (this.checked == true) { chart.type = this.value; } });

    d3.selectAll(id("field-radio input")) // e.g. average, mean, max
      .each(function (d) { if (this.checked == true) { chart.fun = APP.fields[this.value]; } });

    d3.selectAll(id("rot-radio input"))
      .each(function (d) { if (this.checked == true) { chart.rotational = this.value; } });

    d3.selectAll(id("logtype-radio input"))
      .each(function (d) { if (this.checked == true) { chart.log_type = this.value; } });

    if (chart.pcntl === "all") { chart.pcntl = "" } else { chart.pcntl = chart.pcntl + "_"; }
    if (chart.ddir === "all")  { chart.ddir = ""  } else { chart.ddir  = chart.ddir  + "_"; }

    chart.sample = chart.pcntl + chart.ddir + "bin";
    if (chart.ddir === "percentiles") {
      chart.sample = chart.ddir;
    }
  });

  APP.chart("#mid_middle", devs, chart1, chart2);
};

// render the nav, this should only happen once
APP.build_nav = function() {
  // top left is empty for now
  // top middle is graph title, populted in APP.chart()

  // benchmarks on the left / middle immediately left of the graph
  var mid_left = d3.select("#mid_left");
  var benchmarks = mid_left.selectAll(".chart1-benchmark-radio")
    .data(APP.benchmarks.sort())
    .enter()
    .append("div")
      .classed({"radio": true, "chart1-benchmark-radio": true});

  benchmarks.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "benchmark-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  benchmarks.append("label")
    .text(function (d) { return d; });

  // ===========================================================================

  var mid_right = d3.select("#mid_right");

  var rot = mid_right.selectAll(".chart1-rot-radio")
    .data(["All", "HDD", "SSD"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "chart1-rot-radio": true});

  rot.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "rot-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  rot.append("label").text(function (d) { return d });

  mid_right.append("hr"); // ==================================================

  var logtypes = mid_right.selectAll(".chart1-logtype-radio")
    .data(["lat", "bw", "iops"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "chart1-logtype-radio": true});

  logtypes.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "chart1-logtype-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  logtypes.append("label")
    .text(function (d) { return d; });

  mid_right.append("hr"); // ==================================================

  var pcntls = mid_right.selectAll(".pcntl-radio")
    .data(["all", "p1", "p99"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "pcntl-radio": true});

  pcntls.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "pcntl-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  pcntls.append("label")
    .text(function (d) { return d; });

  mid_right.append("hr"); // ==================================================

  // chart type on the bottom left under pcntl
  var types = mid_right.selectAll(".chart1-chart-type-radio")
    .data(["c3.line", "c3.bar", "c3.scatter", "d3.box"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "chart1-chart-type-radio": true});

  types.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "chart-type-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  types.append("label")
    .text(function (d) { return d; });

  mid_right.append("hr"); // ==================================================

  // ddir gets appended after benchmark selection
  // no trim data for now, so leave it off
  var ddirs = mid_right.selectAll(".ddir-radio")
    .data(["all", "read", "write", "pcntl"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "ddir-radio": true});

  ddirs.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "ddir-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  ddirs.append("label")
    .text(function (d) { return d; });

  mid_right.append("hr"); // ==================================================

  var fields = mid_right.selectAll(".chart1-field-radio")
    .data(d3.keys(APP.fields).sort())
    .enter()
    .append("div")
      .classed({"radio-inline": true, "chart1-field-radio": true});

  fields.append("input").attr("type", "radio")
    .property("checked", function (d) { return d === "average"; })
    .attr("name", "chart1-field-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  fields.append("label")
    .text(function (d) { return d; });

  // ==========================================================================

  // devices along the bottom of the graph
  var devs = d3.select("#bot_middle").selectAll(".device-checkbox")
    .data(APP.devices.sort())
    .enter()
    .append("div")
      .classed({"checkbox-inline": true, "device-checkbox": true});

  devs.append("input").attr("type", "checkbox")
    .property("checked", "1")
    .attr("name", function (d) { return d; })
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  devs.append("label")
    .attr("for", function (d) { return d; })
    .html(function (d) { return d; });

  APP.change();
};

// slow & stupid & effective
APP.uniq = function (list, fun, category) {
  var out = [];
  list.forEach(function (d,i) {
    var val = fun(d);

    // temporary hack for older data files that don't have the
    // fio_name field populated
    if (category === "benchmark") {
      var re = new RegExp(list[i].fio_command.device.name + "-(.*)");
      var found = re.exec(list[i].fio_command.name);
      val = list[i].fio_command.fio_name = found[1];
    }

    if (out.indexOf(val) === -1) {
      out.push(val);
    }
  });

  return out.sort();
};

// called on page load to pull data from the server into memory for display/processing
// returns a promise that resolves when all data is loaded
// Usage: APP.run.then(function () { alert("loaded!"); });
APP.run = function () {
  APP.inventory = [];
  APP.summaries = [];

  return new Promise(function (resolve, reject) {
    d3.json("/inventory", function (error, inventory) {
      // failed, log and reject the promise
      if (error) {
        console.log("d3.json error: " + error);
        return reject(error);
      }

      // ignore clat & slat - they're huge and useless
      d3.keys(inventory).filter(function (key) {
        if (key === "clat" || key === "slat") {
          return false;
        }
        return true;
      }).forEach(function (key) {
        APP.inventory = APP.inventory.concat(inventory[key]);
      });

      // load all of the summaries over XHR
      APP.inventory.forEach(function (json_file) {
        d3.json(json_file, function (error, summary) {
          // failed, log and reject the promise
          if (error) {
            console.log("d3.json error: " + error);
            return reject(error);
          }

          APP.summaries.push(summary);

          if (APP.summaries.length === APP.inventory.length) {
            APP.build_indices();

            // when data loading & indexing is complete, resolve the promise
            return resolve();
          }
        });
      });
    });
  });
};

// called after all the data is downloaded and extract some lists for use
// in building the UI ... maybe should be renamed
APP.build_indices = function () {
  console.log("Indexing complete. APP:", APP);

  APP.devices = APP.uniq(APP.summaries, function (d) { return d.fio_command.device.name; });
  APP.benchmarks = APP.uniq(APP.summaries, function (d) { return d.fio_command.fio_name; }, "benchmark");
  APP.suites = APP.uniq(APP.summaries, function (d) { return d.fio_command.suite_name; });

  // assign devices colors at startup so they're consistent across changes
  var colors = d3.scale.category20();
  APP.device_colors = {};
  APP.devices.forEach(function (d,i) {
    APP.device_colors[d] = colors(i);
  });

  // HACK: fix summary.name, check for duplicate entrires
  APP.by_name = {};
  APP.summaries.forEach(function (d) {
    d.name = d.fio_command.name; // the preprocessor isn't setting this correctly, fix later
    if (APP.by_name.hasOwnProperty(d.name)) {
      var old = APP.by_name[d.name];
      //console.log("WARNING DUPLICATE NAME(" + old.path + "): (name, have, found) ", d.name, d, old);
    } else {
      APP.by_name[d.name] = d;
    }
  });

  var sample_types = {};
  APP.summaries.forEach(function (smry) {
    d3.keys(smry).forEach(function (key) {
      if (key.length > 1 && key.match(/bin/)) {
        sample_types[key] = true;
      }
    });
  });
  APP.sample_types = d3.keys(sample_types);
};

// vim: et ts=2 sw=2 ai smarttab
