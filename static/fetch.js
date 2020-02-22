function getBaseUrl() {
    xmlHttp = new XMLHttpRequest();
    xmlHttp.open( "GET", "/baseurl", false ); // false for synchronous request
    xmlHttp.send( null );
    return xmlHttp.responseText;
}

function startFetch() {
    let wsProtocol = "ws:";
    if(window.location.protocol == "https:") {
        wsProtocol = "wss:";
    }
    let ws = new WebSocket(wsProtocol+"//"+getBaseUrl()+"/ws");
    let columns = ["name", "cpu", "mem", "memperc", "net", "block"];
    
    ws.onmessage = function (event) {
        let data = JSON.parse(event.data);
        for (container_data of data) {
            tr_id = container_data.name.substring(1);
            let tr = document.getElementById(tr_id);
            if (tr == null) {
                tr = document.createElement("tr");
                tr.setAttribute("id", tr_id);
        
                for (col of columns) {
                    let td = document.createElement("td");
                    td.setAttribute("class", col);
                    tr.appendChild(td);
                }
                document.getElementById("stats").appendChild(tr);
            }
        
            let td_name = document.querySelector("#" + tr_id + " .name");
            td_name.innerHTML = container_data.name;
            let td_cpu = document.querySelector("#" + tr_id + " .cpu");
            td_cpu.innerHTML = container_data.cpu;
            let td_mem = document.querySelector("#" + tr_id + " .mem");
            td_mem.innerHTML = container_data.memory + " / " + container_data.memoryLimit;
            let td_memperc = document.querySelector("#" + tr_id + " .memperc");
            td_memperc.innerHTML = container_data.memoryPercent;
            let td_net = document.querySelector("#" + tr_id + " .net");
            td_net.innerHTML = container_data.netIn + " / " + container_data.netOut;
            let td_block = document.querySelector("#" + tr_id + " .block");
            td_block.innerHTML = container_data.blockIn + " / " + container_data.blockOut;
        }
    };
}