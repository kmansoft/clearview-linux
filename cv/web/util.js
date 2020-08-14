class HtmlSync {
    constructor(parentId, templateId) {
        this.elParent = document.getElementById(parentId)
        this.elTemplate = document.getElementById(templateId)
    }

    syncWithData(dataList, onclick, updateFunc, rebind) {
        // 1: remove excessive html nodes
        let children = this.elParent.children;
        while (dataList.length < children.length) {
            const node = children[children.length - 1];
            this.elParent.removeChild(node);
            children = this.elParent.children;
        }

        // 2: sync the list creating and updating if necessary
        for (let i in dataList) {
            const data = dataList[i];

            let node;
            if (i < children.length) {
                node = children[i]
            } else {
                node = document.importNode(this.elTemplate.content, true);
                node = this.elParent.appendChild(node.firstElementChild)
            }

            if (node.key != data.key) {
                updateFunc(node, data);
                node.key = data.key;
            } else if (rebind) {
                updateFunc(node, data);
            }
            node.data = data;
            if (onclick) {
                node.onclick = (e) => {
                    e.preventDefault();
                    onclick(data);
                }
            }
        }
    }

    setNodeText(node, text) {
        while (node.firstChild) {
            node.removeChild(node.firstChild)
        }
        node.innerText = text;
    }
}

class ButtonGroup {
    constructor(parentId, onclick, initialId) {
        this.elParentNode = document.getElementById(parentId)
        this.elButtonList = this.elParentNode.querySelectorAll("button")

        if (onclick) {
            for (let s of this.elButtonList) {
                s.onclick = (e) => {
                    e.preventDefault();
                    onclick(e)
                }
            }
        }

        this.selected_id = initialId || this.elButtonList[0].id;
        this.setSelectedIdImpl()
    }

    setSelectedId(id) {
        if (this.selected_id !== id) {
            this.selected_id = id;
            this.setSelectedIdImpl()
        }
    }

    setSelectedIdImpl() {
        for (let s of this.elButtonList) {
            if (s.id === this.selected_id) {
                s.classList.add("btn-secondary");
                s.classList.remove("btn-light")
            } else {
                s.classList.remove("btn-secondary");
                s.classList.add("btn-light")
            }
        }
    }
}

class NavBar {
    constructor(parentId) {
        this.elParentNode = document.getElementById(parentId)
        this.items = [{
            link: "charts-nodes.html",
            text: "ClearView",
            slug: "index",
        }, {
            link: "charts-overview.html",
            text: "Overview",
            slug: "overview",
        }, {
            link: "charts-network.html",
            text: "Network",
            slug: "network",
        }, {
            link: "charts-disk.html",
            text: "Disk",
            slug: "disk",
        }, {
            link: "charts-process.html",
            text: "Process",
            slug: "process",
        }, {
            link: "charts-app-apache.html",
            text: "Apache",
            slug: "apache",
            liid: "nav-bar-apache"
        }, {
            link: "charts-app-nginx.html",
            text: "Nginx",
            slug: "nginx",
            liid: "nav-bar-nginx"
        }, {
            link: "charts-app-mysql.html",
            text: "MySQL",
            slug: "mysql",
            liid: "nav-bar-mysql"
        }, {
            link: "charts-app-pgsql.html",
            text: "PgSQL",
            slug: "pgsql",
            liid: "nav-bar-pgsql"
        }, {
            link: "charts-system.html",
            text: "System",
            slug: "system",
        }];

        this.link = null;
        this.getIsVisible = null;
    }

    getLinkBySlug(slug) {
        for (let item of this.items) {
            if (item.slug === slug) {
                return item.link;
            }
        }
    }

    setIsActive(link) {
        if (this.link !== link) {
            this.link = link;
            window.requestAnimationFrame(() => this.render())
        }
    }

    setOnClick(onclick) {
        if (this.onclick !== onclick) {
            this.onclick = onclick;
            window.requestAnimationFrame(() => this.render())
        }
    }

    setLimit(limit) {
        if (this.limit !== limit) {
            this.limit = limit;
            window.requestAnimationFrame(() => this.render())
        }
    }

    setGetIsVisible(func) {
        if (this.getIsVisible !== func) {
            this.getIsVisible = func;
            window.requestAnimationFrame(() => this.render())
        }
    }

    render() {
        let before = null;
        for (let node = this.elParentNode.firstElementChild; node;) {
            const next = node.nextElementSibling;

            if (node.classList.contains("dropdown-item-right")) {
                if (before == null) {
                    before = node;
                }
            } else {
                node.remove();
            }

            node = next;
        }

        const items = this.limit > 0 ? this.items.slice(0, this.limit) : this.items
        for (let item of items) {
            const elListItem = document.createElement("li");
            const elAnchor = document.createElement("a");
            elListItem.appendChild(elAnchor);
            elListItem.classList.add("nav-item");
            if (item.liid) {
                elListItem.id = item.liid;
                if (this.getIsVisible && !this.getIsVisible(item.liid)) {
                    elListItem.classList = "d-none";
                }
            }

            elAnchor.classList.add("nav-link");
            if (this.link === item.link) {
                elListItem.classList.add("active");
                elAnchor.href = "#";
                elAnchor.onclick = (e) => {
                    e.preventDefault();
                }
            } else {
                if (this.onclick != null) {
                    elAnchor.href = "#";
                    elAnchor.onclick = (e) => {
                        e.preventDefault();
                        this.onclick(item.slug, item.link);
                    }
                } else {
                    elAnchor.href = "./" + item.link;
                }
            }
            elAnchor.appendChild(document.createTextNode(item.text));
            this.elParentNode.insertBefore(elListItem, before);
        }
    }
}

class DiskItem {
    constructor(item) {
        this.key = item.name;
        this.read_ops = item.read_ops;
        this.write_ops = item.write_ops;
        this.space_total = item.space_total;
        this.space_free = item.space_free;
        this.inode_total = item.inode_total;
        this.inode_free = item.inode_free;
    }
}

class ProcessItem {
    constructor(item) {
        this.key = item.name;
        this.user = item.user;
        this.count = item.count;
        this.io_total = item.io_total;
        this.cpu = item.cpu;
        this.memory = item.memory;
    }
}

function humanDataSize(bytes) {
    const thresh = 1024;
    if (bytes === 0) {
        return "0"
    }
    if (Math.abs(bytes) < thresh) {
        return (bytes | 0) + ' B';
    }
    let units = ['KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    let u = -1;
    do {
        bytes /= thresh;
        ++u;
    } while (Math.abs(bytes) >= thresh && u < units.length - 1);
    if (bytes >= 100) {
        return Math.floor(bytes) + ' ' + units[u];
    }
    return bytes.toFixed(bytes >= 10 ? 1 : 2) + ' ' + units[u];
}

function humanDataSizePerSecond(bytes) {
    if (bytes === 0) {
        return "0"
    }
    return humanDataSize(bytes) + '/s';
}

function percentValue(fraction, total) {
    if (total <= 0) {
        return 0
    }
    const p = fraction * 100 / total;
    if (p < 10) {
        return p.toFixed(2) + "%"
    }
    return p.toFixed(1) + "%"
}

function fractionalValue(value) {
    if (value <= 0) {
        return 0
    }
    if (value < 10) {
        return value.toFixed(2)
    }
    if (value < 100) {
        return value.toFixed(1)
    }
    return value | 0
}

function parseDuration(s) {
    if (s.endsWith("d")) {
        return 1440 * (s.substr(0, s.length - 1) | 0)
    }
    if (s.endsWith("h")) {
        return 60 * (s.substr(0, s.length - 1) | 0)
    }
    return s | 0
}

function makeRequestFromMinutes(minutes) {
    let pointCount;
    switch (minutes) {
        case 30:
        case 60:
        case 120:
        case 360:
            pointCount = 30;
            break;
        case 720:
        case 1440:
            pointCount = 24;
            break;
        case 20160: // 2 weeks
            pointCount = 28;
            break;
        case 60480: // 6 weeks
            pointCount = 42;
            break;
        default:
            pointCount = 32;
            break;
    }

    const pointDuration = 60 * minutes / pointCount;

    return {
        point_count: pointCount,
        point_duration: pointDuration
    }
}

const TIME_DURATION_LIST = [
    "30", "60",
    "2h", "6h", "12h",
    "1d", "4d", "14d", "42d",
    "96d", "192d", "384d"
];

function roundValue(value) {
    return Math.floor(value * 1000) / 1000
}
