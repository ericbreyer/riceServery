let appendChild = new Vue({
    el: "#serveryInfo",
    data: {
        text: [],
        allFoods: [],
        serveryFilter: [],
        timeFilter: "All",
        daysFilter: "All",
        toRate: "",
        rating: "0",
        sidebarActive: true,
        loading: true,
    },
    methods: {
        currentDate: function() {
            today = new Date()
            switch (today.getDay()) {
                case 0:
                    this.daysFilter = "Sunday"
                    break
                case 1:
                    this.daysFilter = "Monday"
                    break
                case 2:
                    this.daysFilter = "Tuesday"
                    break
                case 3:
                    this.daysFilter = "Wednesday"
                    break
                case 4:
                    this.daysFilter = "Thursday"
                    break
                case 5:
                    this.daysFilter = "Friday"
                    break
                case 6:
                    this.daysFilter = "Saturday"
                    break
            }
            if (today.getHours() >= 14) {
                this.timeFilter = "dinner"
            }
            else {
                this.timeFilter = "lunch"
            }
        },
        changeText: function(text) {
            console.log(this.text)
            this.text = text
            this.allFoods = []
            text.forEach(e1 => {
                JSON.parse(e1).MealTimeGroups.forEach(e2 => {
                    e2.MealDayGroups.forEach(e3 => {
                        if(e3.Meals == null) {
                            return
                        }
                        e3.Meals.forEach(e4 => {
                            this.allFoods.push({
                                "Name": e4.Name,
                                "Day": e3.Name,
                                "Time": e2.Name,
                                "Servery": JSON.parse(e1).Name
                            })
                        })
                    })
                })
            });
            //console.log(this.text)
        },

        fetchData: function() {
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
        },
        sendRating: function() {

            if(this.rating == "0") {
                alert("Invalid Rating")
                return
            }

            var xhttp = new XMLHttpRequest();

            fetchData = this.fetchData

            xhttp.onreadystatechange = function() {
                if (this.readyState == 4 && this.status == 200) {
                    fetchData()
                }
            };
            xhttp.open("POST", "/updateRating", false);

            data = JSON.stringify({ "Name" : this.toRate, "Rating" : parseInt(this.rating)})

            xhttp.send(data);

            this.toRate = ""
            this.hidePopUp();

            this.rating = "0"
        },

        rateFood: function(servery, time, day, food, e) {
            foodObj = JSON.parse(this.text[servery]).MealTimeGroups[time].MealDayGroups[day].Meals[food]
            console.log(foodObj)
            console.log(e.target.value)
            Vue.set(foodObj, "Name", foodObj.Name + "a")
            foodObj.Name = "aboicb"
            foodObj = JSON.parse(this.text[servery]).MealTimeGroups[time].MealDayGroups[day].Meals[food]
            console.log(foodObj)
            this.hidePopUp()
            
        },

        movePopUp: function(e) {
            this.$refs.popUp.style.display = "flex"
            var x = e.clientX; 
            var y = e.clientY;

            scrollOffset = -this.$refs.mainBody.getBoundingClientRect().y
            width = document.body.clientWidth

            xpos = x+2

            if (xpos +172 > width) {
                xpos = width-172
            }

            console.log ({
                "x":x,
                "width":width,
                "xpos":xpos
            })

            this.$refs.popUp.style.marginLeft  = xpos+"px";
            this.$refs.popUp.style.marginTop  =  (y+scrollOffset+2)+"px";
        },

        hidePopUp: function() {
            this.$refs.popUp.style.display = "none"
        },


    },
    created: function() {
        this.fetchData()
        setInterval(this.fetchData, 60 * 1000)
        this.loading = false
        this.currentDate()
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
  