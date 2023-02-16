# riceServery

Personally redesigning the [Rice Dining website](https://dining.rice.edu/) to be more immediately useable and feature-rich.

Live website is hosted at digital ocean using docker and can be found at https://servery.ericbreyer.com

The backend is written in go. It fetches and stores the current menu data on an interval, and passes it along to the client on request. The frontend is written in plain html/css, and the javascript uses some vue.js to dynamically load content.
