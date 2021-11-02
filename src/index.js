let appendChild = new Vue({
    el: "#serveryInfo",
    data: {
        text: [],
        serveyFilter: "All",
        timeFilter: "All",
        daysFilter: "All"
    },
    methods: {
        changeText: function(text) {
            console.log(this.text)
            this.text = text
            console.log(this.text)
        }
    },
    created: function() {
        var xhttp = new XMLHttpRequest();
        var text
        xhttp.onreadystatechange = function() {
            if (this.readyState == 4 && this.status == 200) {

                text = JSON.parse(this.response).Jsons
            }
        };
        xhttp.open("GET", "/data", false);
        xhttp.send();
        this.changeText(text)
    }
})

/*    const status = node.isOnline() ? 'online' : 'offline'
    
    console.log(`Node status: ${status}`)
    document.getElementById('status').innerHTML = `<object data="logo.svg" type="image/svg+xml" style="height: 100px;">
    </object> Node status: ${status}`
    var link = document.querySelector("link[rel~='icon']");
      if (!link) {
          link = document.createElement('link');
          link.rel = 'icon';
          document.getElementsByTagName('head')[0].appendChild(link);
      }
      link.href = 'logo.svg';
} */
  