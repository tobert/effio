// Slide & slide data definitions. I load this last so I can render
// graphs and so on from here.

var SLIDES = {};

SLIDES["000 Title Page"] = function () {
  // Datastax / Cassandra Summit / etc.
};

// f7u12 guy sleeping in a bed, when the pager goes off ...
SLIDES["001 It was a dark and quiet night ..."] = function () {
  // load image in mid div
};

// an sms conversation on screen using imessage.js
SLIDES["002 Rageguy"] = function () {
  // TODO: add times
  var conversation = [
    [ "recv", "Naggy OS", "ALERT: cassandra13.prod SLA violation READ took > 1 second" ],
    [ "recv", "Naggy OS", "ALERT: cassandra13.prod Service recovered" ],
    [ "recv", "Boss",     "uh, is Cassandra down!?" ],
    [ "send", "Boss",     "No. We'll talk in the morning. Good night." ],
    [ "recv", "Boss",     "what about sla?" ],
    [ "send", "Boss",     "What about it?" ],
    [ "recv", "Boss",     "u said it would be fine" ],
    [ "send", "Boss",     "No. I said it would work and that these things would happen." ],
    [ "recv", "Boss",     "Are you still going on about the SSDs?" ],
    [ "send", "Boss",     "Yes. Yes I am. Good night!" ]
  ];
};

// picture of a rage face in a bed sleeping
SLIDES["003 Everything OK Guy"] = function () {
};
