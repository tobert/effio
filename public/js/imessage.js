var imessage = {
  months: ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"],

  send: function (message, icon) {
    var conv = d3.select(".conversation");
    if (icon) {
      conv.append("img").attr("src", icon).attr("class", "icon-sent");
    }
    var msg = conv.append("div").attr("class", "msg sent");

    msg.append("div").attr("class", "reflect");
    msg.append("p").text(message);
    conv.node().scrollTop = conv.node().scrollHeight;
  },

  recv: function (message, icon) {
    var conv = d3.select(".conversation");
    if (icon) {
      conv.append("img").attr("src", icon).attr("class", "icon-received");
    }
    var msg = conv.append("div").attr("class", "msg received");

    msg.append("div").attr("class", "reflect");
    msg.append("p").text(message);
    conv.node().scrollTop = conv.node().scrollHeight;
  },

  timestamp: function (date) {
    var ampm = "PM";
    var hr = date.getHours();
    if (hr < 12) { ampm = "AM"; }
    if (hr == 0) { hr = 12; }
    if (hr > 12) { hr = hr - 12; }

    var min = date.getMinutes();
    if (min.length == 1) {
      min = "0" + min;
    }

    var str = imessage.months[date.getMonth()] + " "
      + date.getDate() + ", " + date.getFullYear() + " "
      + hr + ":" + min + " " + ampm;

    d3.select(".conversation")
      .append("div").attr("class", "time")
      .append("p").text(str);
  },

  clear: function () {
    var conv = d3.select(".conversation");
    conv.selectAll("img").remove();
    conv.selectAll("div").remove();
  }
};

