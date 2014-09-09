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

APP.main = function () {
  console.log("APP:", APP);

  APP.devices = APP.uniq(APP.summaries, function (d) { return d.fio_command.device.name; });
  APP.benchmarks = APP.uniq(APP.summaries, function (d) { return d.fio_command.fio_name; }, "benchmark");
  APP.suites = APP.uniq(APP.summaries, function (d) { return d.fio_command.suite_name; });

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

APP.render_chart = function (target) {
  APP.setup_chart_controls(target);
  APP.build_nav();
};

APP.chart = function (target, benchmark, sample_type, devices, chart_type, rot, fun) {
  console.log("APP.chart(", benchmark, sample_type, devices, chart_type, fun, ")");

  d3.select("#top_mid").text(benchmark + " / " + sample_type);

  // finds the summaries that contain the benchmark requested
  var summaries = APP.summaries
    // only display the selected benchmark name
    .filter(function (d) { return d.fio_command.fio_name === benchmark; })
    // sort by device name to keep layout consistent
    .sort(function (a,b) {
      if (a.fio_command.device.name > b.fio_command.device.name) { return  1; }
      if (a.fio_command.device.name < b.fio_command.device.name) { return -1; }
      return 0;
    })
    // only display selected devices
    .filter(function (d) { return devices.hasOwnProperty(d.fio_command.device.name); })
    // quick split on ssd/hdd
    .filter(function (d) {
      if (rot === "All") {
        return true;
      } else {
        return d.fio_command.device.rotational === (rot === "HDD");
      }
    });

  console.log("Selected summaries", summaries);

  var ctype = chart_type.split(".");
  if (ctype[0] === "c3") {
    APP.c3chart(target, summaries, sample_type, ctype[1], fun);
  } else if (ctype[0] === "d3" && ctype[1] === "box") {
    APP.d3box(target, summaries, sample_type, fun);
  } else {
    alert("Invalid chart type: '" + chart_type + "'");
  }
};

APP.c3chart = function (target, data, sample_type, chart_type, fun) {
  console.log("APP.c3chart", data, sample_type, chart_type, fun);
  // format the data for C3
  var cols = data.map(function (summary) {
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

  return c3.generate({
    bindto: target,
    data: { columns: cols, type: chart_type },
    axis: {
      y: { label: { text: "Latency (usec)", position: "outer-middle" } },
      x: { label: { text: "Time Offset (seconds)" } }
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

APP.run = function () {
  APP.inventory = [];
  APP.summaries = [];

  d3.json("/inventory", function (error, inventory) {
    if (error) { return alert(error); }

    // ignore clat & slat - they're huge and useless
    d3.keys(inventory).filter(function (key) {
      //if (key === "lat" || key === "bw" || key === "iops") {
      if (key === "lat") {
        return true;
      }
      return false;
    }).forEach(function (key) {
      APP.inventory = APP.inventory.concat(inventory[key]);
    });

    APP.inventory.forEach(function (json_file) {
      d3.json(json_file, function (error, summary) {
        if (error) { return alert(error); }

        APP.summaries.push(summary);

        // fire the main program once all data is downloaded
        if (APP.summaries.length === APP.inventory.length) {
          APP.main();
        }
      });
    });
  });
};

APP.setup_chart_controls = function (target) {
  var body = d3.select(target);
  body.selectAll("div").remove();
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
APP.change = function () {
  var devs = {};
  var benchmark, ddir, pcntl, chart_type, field, rot;

  d3.selectAll(".device-checkbox input")
    .each(function (d) { if (this.checked == true) { devs[this.value] = true; } });

  d3.selectAll(".benchmark-radio input")
    .each(function (d) { if (this.checked == true) { benchmark = this.value; } });

  d3.selectAll(".ddir-radio input")
    .each(function (d) { if (this.checked == true) { ddir = this.value; } });

  d3.selectAll(".pcntl-radio input")
    .each(function (d) { if (this.checked == true) { pcntl = this.value; } });

  d3.selectAll(".chart-type-radio input")
    .each(function (d) { if (this.checked == true) { chart_type = this.value; } });

  d3.selectAll(".field-radio input")
    .each(function (d) { if (this.checked == true) { field = this.value; } });

  d3.selectAll(".rot-radio input")
    .each(function (d) { if (this.checked == true) { rot = this.value; } });

  if (pcntl === "all") { pcntl = "" } else { pcntl = pcntl + "_"; }
  if (ddir === "all")  { ddir = ""  } else { ddir = ddir + "_"; }

  var sample_type = pcntl + ddir + "bin";
  if (ddir === "percentiles") {
    sample_type = ddir;
  }

  console.log("APP.change -> APP.chart(", benchmark, sample_type, devs, chart_type, rot, APP.fields[field], ");");
  APP.chart("#mid_middle", benchmark, sample_type, devs, chart_type, rot, APP.fields[field]);
};

// render the nav, this should only happen once
APP.build_nav = function() {
  // TODO: add y2data rendering

  // top left is empty for now

  // top middle is graph title, populted in APP.chart()

  var rot = d3.select("#top_right").selectAll(".rot-radio")
    .data(["All", "HDD", "SSD"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "rot-radio": true});

  rot.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "rot-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  rot.append("label").text(function (d) { return d });

  // benchmarks on the left / middle immediately left of the graph
  var mid_left = d3.select("#mid_left");
  var benchmarks = mid_left.selectAll(".benchmark-radio")
    .data(APP.benchmarks.sort())
    .enter()
    .append("div")
      .classed({"radio": true, "benchmark-radio": true});

  benchmarks.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "benchmark-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  benchmarks.append("label")
    .text(function (d) { return d; });

  mid_left.append("hr");

  // ddir gets appended after benchmark selection
  // to allow select of left/read & right/write etc. once y2data is implemented
  var ddirs = mid_left.selectAll(".ddir-radio")
    .data(["all", "read", "write", "trim", "percentiles"])
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

  // a second set of benchmark selectors, but these display iops & bw
  var mid_right = d3.select("#mid_right");
  var benchmarks2 = mid_right.selectAll(".benchmark2-radio")
    .data(APP.benchmarks.sort())
    .enter()
    .append("div")
      .classed({"radio": true, "benchmark2-radio": true});

  benchmarks2.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "benchmark2-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  benchmarks2.append("label")
    .text(function (d) { return d; });

  var logtypes = mid_right.selectAll(".logtype-radio")
    .data(["lat", "bw", "iops"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "logtype-radio": true});

  logtypes.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "logtype-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  logtypes.append("label")
    .text(function (d) { return d; });

  var bot_left = d3.select("#bot_left");

  // same as with ddir but on the bottom left
  var pcntls = bot_left.selectAll(".pcntl-radio")
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

  bot_left.append("br");

  // chart type on the bottom left under pcntl
  var types = bot_left.selectAll(".chart-type-radio")
    .data(["c3.line", "c3.bar", "c3.scatter", "d3.box"])
    .enter()
    .append("div")
      .classed({"radio-inline": true, "chart-type-radio": true});

  types.append("input").attr("type", "radio")
    .property("checked", function (d,i) { return i === 0; })
    .attr("name", "chart-type-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  types.append("label")
    .text(function (d) { return d; });

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
    .text(function (d) { return d; });

  var bot_right = d3.select("#bot_right");
  var fields = bot_right.selectAll(".field-radio")
    .data(d3.keys(APP.fields).sort())
    .enter()
    .append("div")
      .classed({"radio-inline": true, "field-radio": true});

  fields.append("input").attr("type", "radio")
    .property("checked", function (d) { return d === "average"; })
    .attr("name", "field-radio")
    .attr("value", function (d) { return d; })
    .on("change", function () { APP.change(); });

  fields.append("label")
    .text(function (d) { return d; });

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
  return out;
};

$(APP.run)

// vim: et ts=2 sw=2 ai smarttab
