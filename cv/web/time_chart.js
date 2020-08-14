'use strict';

class Animator {
    getTime() {
        return window.performance.now()
    }

    clamp(value, min, max) {
        if (value < min) {
            return min
        } else if (value > max) {
            return max
        } else {
            return value
        }
    }
}

class PointMoveAnimator extends Animator {
    static attach(animator, x, y, func, time) {
        if (!animator) {
            animator = new PointMoveAnimator(x, y, func, time)
        } else if (!animator.setNext(x, y)) {
            animator.update()
        }

        return animator
    }

    constructor(x, y, func, time) {
        super();

        this.currX = this.fromX = this.nextX = x;
        this.currY = this.fromY = this.nextY = y;
        this.currT = this.fromT = 0;
        this.func = func;
        this.time = time || 200
    }

    setNext(x, y) {
        if (this.nextX !== x || this.nextY !== y) {
            this.fromX = this.currX;
            this.fromY = this.currY;
            this.fromT = this.getTime();
            this.nextX = x;
            this.nextY = y;

            requestAnimationFrame(() => {
                this.update()
            });

            return true
        } else {
            return false
        }
    }

    update() {
        if (this.fromT !== 0) {
            this.currT = this.getTime();

            const factor = (this.currT - this.fromT) / this.time;
            if (factor < 1.0) {
                this.currX = (this.nextX - this.fromX) * factor + this.fromX;
                this.currY = (this.nextY - this.fromY) * factor + this.fromY;
            } else {
                this.currX = this.fromX = this.nextX;
                this.currY = this.fromY = this.nextY;
                this.fromT = 0;
            }

            (this.func)(this.currX, this.currY);

            if (factor < 1.0) {
                requestAnimationFrame(() => {
                    this.update()
                })
            }
        }
    }

    current(forceFinish) {
        if (forceFinish) {
            this.currX = this.fromX = this.nextX;
            this.currY = this.fromY = this.nextY;
            this.fromT = 0;
        }

        (this.func)(this.currX, this.currY)
    }

    getCurrX() {
        return this.currX;
    }

    getCurrY() {
        return this.currY
    }
}

class ShowHideAnimator extends Animator {
    static attach(animator, func, time) {
        if (animator == null) {
            animator = new ShowHideAnimator(func, time)
        }
        return animator
    }

    constructor(func, time) {
        super();

        this.first = true;
        this.func = func
        this.time = time || 200
    }

    setVisible(visible) {
        const value = visible ? 1.0 : 0.0;
        if (this.first) {
            this.first = false
            this.currV = this.fromV = this.nextV = value;
            this.currT = this.fromT = 0;

            (this.func)(this.currV);

            return true
        } else if (this.nextV !== value) {
            this.fromV = this.currV;
            this.fromT = this.getTime();
            this.nextV = value;

            requestAnimationFrame(() => {
                this.update()
            });

            return true
        } else {
            return false
        }
    }

    update() {
        if (this.fromT !== 0) {
            this.currT = this.getTime();

            const factor = (this.currT - this.fromT) / this.time;
            if (factor < 1.0) {
                this.currV = (this.nextV - this.fromV) * factor + this.fromV;
            } else {
                this.currV = this.fromV = this.nextV;
                this.fromT = 0;
            }

            this.currV = this.clamp(this.currV, 0.0, 1.0);

            (this.func)(this.currV);

            if (factor < 1.0) {
                requestAnimationFrame(() => {
                    this.update()
                })
            }
        }
    }
}

class SvgDraw {
    constructor(elSvg) {
        this.elSvg = elSvg;
        this.elCurr = elSvg;
    }

    clear() {
        while (this.elSvg.firstChild) {
            this.elSvg.firstChild.remove()
        }
    }

    line(x1, y1, x2, y2, stroke, strokeWidth) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "line");

        node.setAttribute("x1", x1);
        node.setAttribute("y1", y1);
        node.setAttribute("x2", x2);
        node.setAttribute("y2", y2);
        node.setAttribute("stroke", stroke);
        node.setAttribute("stroke-width", strokeWidth || "1px");

        this.elCurr.appendChild(node);
    }

    rect(x, y, width, height, fill, stroke) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "rect");

        node.setAttribute("x", x);
        node.setAttribute("y", y);
        node.setAttribute("width", width);
        node.setAttribute("height", height);
        node.setAttribute("fill", fill);
        node.setAttribute("stroke", stroke);

        this.elCurr.appendChild(node);
    }

    text(x, y, className, s, fill) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "text");

        node.setAttribute("x", x);
        node.setAttribute("y", y);
        node.setAttribute("fill", fill || "#202020");
        if (className) {
            node.classList.add(className);
        }

        const text = document.createTextNode(s)
        node.appendChild(text)

        this.elCurr.appendChild(node);
    }

    textRight(x, y, className, s, fill) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "text");

        node.setAttribute("x", x);
        node.setAttribute("y", y);
        node.setAttribute("fill", fill || "#202020");
        node.setAttribute("text-anchor", "end");
        if (className) {
            node.classList.add(className);
        }

        const text = document.createTextNode(s)
        node.appendChild(text)

        this.elCurr.appendChild(node);
    }

    polygon(points, fill) {
        let pt = "";
        for (let i = 0; i < points.length; i++) {
            if (i != 0) {
                pt = pt + " ";
            }
            pt = pt + points[i].x;
            pt = pt + ",";
            pt = pt + points[i].y;
        }

        const node = document.createElementNS("http://www.w3.org/2000/svg", "polygon");
        node.setAttribute("points", pt);
        node.setAttribute("fill", fill);
        node.setAttribute("stroke", "none");

        this.elCurr.appendChild(node);
    }

    polyline(points, stroke, strokeWidth) {
        let pt = "";
        for (let i = 0; i < points.length; i++) {
            if (i != 0) {
                pt = pt + " ";
            }
            pt = pt + points[i].x;
            pt = pt + ",";
            pt = pt + points[i].y;
        }

        const node = document.createElementNS("http://www.w3.org/2000/svg", "polyline");
        node.setAttribute("points", pt);
        node.setAttribute("fill", "none");
        node.setAttribute("stroke", stroke);
        node.setAttribute("stroke-width", strokeWidth || "1px");

        this.elCurr.appendChild(node);
    }

    circle(x, y, radius, fill) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "circle");

        node.setAttribute("cx", x);
        node.setAttribute("cy", y);
        node.setAttribute("r", radius);
        node.setAttribute("fill", fill);
        node.setAttribute("stroke", "none");

        this.elCurr.appendChild(node);
    }

    beginTransform(translateX, translateY, rotate) {
        const node = document.createElementNS("http://www.w3.org/2000/svg", "g");

        node.setAttribute("transform",
            "translate(" + translateX + "," + translateY + ") rotate(" + rotate + ")");

        this.elCurr.appendChild(node);
        this.elCurr = node;
    }

    endTransform() {
        this.elCurr = this.elCurr.parentNode;
    }
}

class TimeValueData {
    static prepareSeriesLabels(data) {
        return data.points.map((v) => {
            return v.t
        });
    }

    static prepareSeries(request, series, sub) {
        for (let s of series || []) {
            if (s.sub === sub) {
                return s.points.map((v) => {
                    return {t: v.t, v: v.n ? null : v.v}
                });
            }
        }

        let endTime = request.end_time
        if (endTime < 0) {
            endTime = Date.time() / 1000;
        }
        let startTime = endTime - request.point_count * request.point_duration;
        startTime = ((startTime / 60) | 0) * 60;
        const out = [];
        for (let i = 0; i <= request.point_count; ++i) {
            out.push({
                t: startTime + i * request.point_duration,
                v: null
            })
        }

        return out
    }

    static prepareSeriesDiff(from, what) {
        console.log("prepareSeriesDiff", from, what)

        const out = [];
        for (let i = 0; i < from.length; ++i) {
            const f = from[i];
            const w = what[i];
            const v = {t: f.t};
            if (f.v == null || w.v == null) {
                v.v = null;
            } else {
                const d = f.v - w.v;
                if (d >= 0.0) {
                    v.v = d;
                } else {
                    v.v = 0.0;
                }
            }
            out.push(v);
        }

        console.log("prepareSeriesDiff ->", out)

        return out
    }

    static getMaxValue(data) {
        console.log("getMaxValue", data)

        let max = null;
        for (let p of data) {
            if (p.v != null) {
                if (max == null || max < p.v) {
                    max = p.v;
                }
            }
        }
        return max;
    }
}

class TimeValueChart {
    constructor(elSvg) {
        this.Mapper = function (fromMin, fromMax, toMin, toMax, flipped) {
            if (fromMax < fromMin + 0.001) {
                fromMax = fromMin + 0.001
            }

            if (flipped) {
                const f = function (value) {
                    const r = toMax - (value - fromMin) * (toMax - toMin) / (fromMax - fromMin);
                    // console.log('Mapper', value, '->', r);
                    return r
                };
                return f
            } else {
                const f = function (value) {
                    const r = toMin + (value - fromMin) * (toMax - toMin) / (fromMax - fromMin);
                    // console.log('Mapper', value, '->', r);
                    return r
                };
                f.inverse = function (r) {
                    const value = fromMin + (r - toMin) * (fromMax - fromMin) / (toMax - toMin)
                    // console.log('Mapper', r, '<-', value);
                    return value
                };
                return f
            }
        };

        this.LabelFormatter = function (series_label) {
            const date = new Date();
            return function (i) {
                date.setTime(series_label[i] * 1000);

                const duration = series_label[1] - series_label[0];
                if (duration >= 8 * 1440) {
                    return date.toLocaleDateString();
                }

                return date.toLocaleTimeString();
            }
        };

        this.elParent = elSvg.parentNode;
        this.elSvg = elSvg;

        elSvg.chart = this;

        this.width = 740;
        this.height = 260;

        this.chart_options = {
            line: '#c0c0c0',
            tick: '#404040'
        };
        this.data_options = [{
            name: 'series 1',
            fill: '#90CAF9c0',
            line: '#2196F3'
        }];

        this.createOverlays();

        this.elEvents.addEventListener("mousemove", (e) => {
            // console.log("mousemove", e.layerX, e.layerY);
            if (this.hl_index_from >= 0) {
                if (this.mapperX) {
                    let index = this.mapperX.inverse(e.layerX);
                    console.log("index in mousemove", index);
                    if (index < 0 || index >= this.series_label.length - 1) {
                        index = -1;
                        this.pointMoveAnimatorText = null;
                        this.pointMoveAnimatorLine = null;
                    }
                    this.setShowSelectTo(index);
                }
            }

            this.onMouseMoveImpl(e.layerX, e.layerY)
        });

        this.elEvents.addEventListener("mouseleave", (e) => {
            // console.log("mouseleave");
            this.onMouseLeaveImpl(e.relatedTarget);
        });

        this.elEvents.addEventListener("mousedown", (e) => {
            console.log("mousedown", e);
            if (this.mapperX) {
                let index = this.mapperX.inverse(e.layerX);
                if (index < 1.0 || index >= this.series_label.length) {
                    index = -1
                }

                if (this.callbacks && this.callbacks.onZoomInSelection) {
                    this.setShowSelectFrom(index);
                }
            }
        });

        this.elEvents.addEventListener("mouseup", (e) => {
            console.log("mouseup", e);

            if (this.callbacks && this.callbacks.onZoomInSelection) {
                if (this.series_label) {
                    if (this.hl_index_from >= 0 && this.hl_index_to >= 0) {
                        let from = Math.round(this.hl_index_from) | 0;
                        let to = Math.round(this.hl_index_to) | 0;
                        if (from > to) {
                            const t = from;
                            from = to;
                            to = t;
                        }
                        console.log("zoom in selection", from, to);
                        this.callbacks.onZoomInSelection(
                            this.series_label, from, to);
                    }
                }
            }
            this.setShowSelectFrom(-1);
            this.setShowSelectTo(-1);
        });

        this.showHideAnimatorZoomFunc = (value) => {
            const elZoomSelect = this.elZoomSelect;

            if (value > 0.0) {
                elZoomSelect.style.visibility = 'visible';
                elZoomSelect.style.opacity = value;
            } else {
                elZoomSelect.style.visibility = 'hidden';
            }
        };

        this.showHideAnimatorLegendFunc = (value) => {
            const elLegendText = this.elLegendText;
            const elLegendLine = this.elLegendLine;

            if (value > 0.0) {
                elLegendText.style.visibility = 'visible';
                elLegendText.style.opacity = value;
                elLegendLine.style.visibility = 'visible';
                elLegendLine.style.opacity = value;
            } else {
                elLegendText.style.visibility = 'hidden';
                elLegendLine.style.visibility = 'hidden';
            }
        };

        this.pointMoveAnimatorTextFunc = (x, y) => {
            const elLegendText = this.elLegendText;

            elLegendText.style.left = x + "px";
            elLegendText.style.top = y + "px";
        };

        this.pointMoveAnimatorLineFunc = (x, y) => {
            const elLegendLine = this.elLegendLine;

            elLegendLine.style.left = x + "px";
            elLegendLine.style.top = y + "px";
        };
    }

    setSize(width, height) {
        this.width = width;
        this.height = height
        this.elSvg.style.width = this.width;
        this.elSvg.style.height = this.height;
    }

    setChartTitle(title) {
        this.chart_title = title
    }

    setCallbacks(callbacks) {
        this.callbacks = callbacks;
    }

    setDataOptions(options) {
        this.data_options = options;

        for (let o of this.data_options) {
            if (o.fill.length > 7 && o.fill.startsWith("#")) {
                o.fill = tinycolor(o.fill).toRgbString()
            }
        }
    }

    setData(series_label, ...series_data) {
        this.series_label = series_label;
        this.series_data = series_data;
    }

    setSeries(...series) {
        if (series == null || series.length === 0) {
            this.series_label = null;
            this.series_data = null;
            return;
        }

        this.series_label = series[0].map((p) => {
            return p.t;
        });
        this.series_data = [];
        for (let s of series) {
            const d = s.map((p) => {
                return p.v;
            });
            this.series_data.push(d);
        }
    }

    setDataMaxY(value) {
        this.forceMaxYFlag = true;
        this.forceMaxYValue = value;
    }

    setShowLegend(index) {
        this.hl_index_x_next = index;
        requestAnimationFrame(() => {
            this.updateOverlays()
        })
    }

    setValueFormatter(formatter) {
        this.valueFormatter = formatter;
    }

    setShowSelectFrom(from) {
        this.hl_index_from_next = from;
        requestAnimationFrame(() => {
            this.updateOverlays()
        })
    }

    setShowSelectTo(to) {
        this.hl_index_to_next = to;
        requestAnimationFrame(() => {
            this.updateOverlays()
        })
    }

    updateOverlays() {
        if (this.hl_index_x === this.hl_index_x_next &&
            this.hl_index_from === this.hl_index_from_next &&
            this.hl_index_to === this.hl_index_to_next) {
            return;
        }

        if (this.mapperX == null) {
            return;
        }

        // Highlighted indexLegend
        const isShowingLegendNow = this.hl_index_x < 0 && this.hl_index_x_next >= 0 ||
            this.hl_index_from >= 0 && this.hl_index_to >= 0 &&
            this.hl_index_from_next < 0 && this.hl_index_to_next < 0;
        this.hl_index_x = this.hl_index_x_next;

        const indexLegend = this.hl_index_x;

        // From/to selection for zooming in
        this.hl_index_from = this.hl_index_from_next;
        this.hl_index_to = this.hl_index_to_next;

        const indexSelectFrom = this.hl_index_from;
        const indexSelectTo = this.hl_index_to;

        const elLegendText = this.elLegendText;
        const elLegendLine = this.elLegendLine;

        this.showHideAnimatorZoom = ShowHideAnimator.attach(
            this.showHideAnimatorZoom,
            this.showHideAnimatorZoomFunc, 350);

        this.showHideAnimatorLegend = ShowHideAnimator.attach(
            this.showHideAnimatorLegend,
            this.showHideAnimatorLegendFunc, 350);

        if (indexSelectFrom >= 0 && indexSelectTo >= 0) {
            this.showHideAnimatorZoom.setVisible(true);
            this.showHideAnimatorLegend.setVisible(false);

            const elZoomSelect = this.elZoomSelect;
            const xSelectFrom = this.mapperX(indexSelectFrom);
            const xSelectTo = this.mapperX(indexSelectTo);

            console.log("xSelect from/to", xSelectFrom, xSelectTo);

            const xSelectMin = Math.min(xSelectFrom, xSelectTo)
            const xSelectMax = Math.max(xSelectFrom, xSelectTo)

            elZoomSelect.style.left = xSelectMin + "px";
            elZoomSelect.style.top = 42 + "px";
            elZoomSelect.style.width = (xSelectMax - xSelectMin) + "px";
            elZoomSelect.style.height = (this.height - 64 - 42) + "px";
        } else if (indexLegend < 0) {
            this.showHideAnimatorZoom.setVisible(false);
            this.showHideAnimatorLegend.setVisible(false);
        } else {
            this.showHideAnimatorZoom.setVisible(false);
            this.showHideAnimatorLegend.setVisible(true);

            const xPos = this.mapperX(indexLegend);

            // legend
            const elLegendTextDraw = new SvgDraw(elLegendText);
            elLegendTextDraw.clear();

            let xpos = 0, ypos = 0, maxwidth = 0;

            for (let i in this.data_options) {
                elLegendTextDraw.rect(xpos + 4, ypos + 4, 32, 16, this.data_options[i].fill, "none")
                elLegendTextDraw.text(xpos + 40, ypos + 16, "legend-text-label", this.data_options[i].name)

                const raw = this.series_data[i][indexLegend];
                const value =
                    (raw != null && this.valueFormatter != null)
                        ? this.valueFormatter(raw)
                        : this.roundValue(raw);

                elLegendTextDraw.text(xpos + 104, ypos + 16, "legend-text-label", value, this.data_options.tick)

                ypos += 24
            }

            elLegendText.style.width = 170 + "px";
            elLegendText.style.height = ypos + "px";

            const xNextText = (xPos + ((xPos < 200) ? 8 : -112 - maxwidth));
            const yNextText = 24;

            this.pointMoveAnimatorText = PointMoveAnimator.attach(
                this.pointMoveAnimatorText, xNextText, yNextText,
                this.pointMoveAnimatorTextFunc, 350);

            this.pointMoveAnimatorText.current(isShowingLegendNow)

            // line
            const elLegendLineDraw = new SvgDraw(elLegendLine);
            elLegendLineDraw.clear();
            elLegendLineDraw.line(4, 0, 4, this.height, "#202020a0")

            elLegendLine.style.width = 8 + 'px';
            elLegendLine.style.height = (this.height - 76) + 'px'

            const xNextLine = xPos - 4;
            const yNextLine = 24;

            this.pointMoveAnimatorLine = PointMoveAnimator.attach(
                this.pointMoveAnimatorLine, xNextLine, yNextLine,
                this.pointMoveAnimatorLineFunc, 150);

            this.pointMoveAnimatorLine.current(isShowingLegendNow)
        }
    }

    onMouseMoveImpl(x, y) {
        if (this.mapperX) {
            let index = Math.round(this.mapperX.inverse(x)) | 0;
            if (index < 0 || index >= this.series_label.length) {
                index = -1
            }

            const svgList = document.querySelectorAll("svg");
            for (let svg of svgList) {
                if (svg.chart) {
                    svg.chart.setShowLegend(index)
                }
            }
        }
    }

    onMouseLeaveImpl(related) {
        while (related) {
            if (related === this.elParent) {
                return
            }
            related = related.parentNode
        }

        if (this.mapperX) {
            const svgList = document.querySelectorAll("svg");
            for (let svg of svgList) {
                if (svg.chart) {
                    svg.chart.setShowLegend(-1);
                    svg.chart.setShowSelectFrom(-1);
                    svg.chart.setShowSelectTo(-1);
                }
            }
        }
    }

    createOverlays() {
        // Zoom in on selection
        const elZoomSelect = this.elZoomSelect = document.createElement("span")
        elZoomSelect.classList.add("zoom-select");

        this.elParent.appendChild(elZoomSelect)

        // Legend: line
        const elLegendLine = this.elLegendLine = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        elLegendLine.classList.add("legend");
        elLegendLine.classList.add("legend-line");

        this.elParent.appendChild(elLegendLine)

        // Legend: text
        const elLegendText = this.elLegendText = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        elLegendText.classList.add("legend");
        elLegendText.classList.add("legend-text");

        this.elParent.appendChild(elLegendText)

        // Events handling a whole big overlay
        const elEvents = this.elEvents = document.createElement("span")
        elEvents.classList.add("events");

        this.elParent.appendChild(elEvents);
    }

    render() {
        const svgDraw = new SvgDraw(this.elSvg);
        svgDraw.clear();

        // Font, use smaller size when it's small
        const tickFontSize = this.getTickFontSize();

        // Font size suffix
        const textClassSuffix = (this.width < 500) ? "small" : "large";

        // Draw the legend
        let xLegendPos = this.width - 8,
            yLegendPos = 24;

        svgDraw.textRight(xLegendPos, yLegendPos, "legend-chart-label-" + textClassSuffix, this.chart_title);

        if (this.series_data == null || this.series_data.length === 0) {
            console.log('No data, nothing to render');
            svgDraw.textRight(xLegendPos, yLegendPos + 20, "legend-chart-label-" + textClassSuffix, "No data")
            return;
        }

        // Legend
        const yLegendSize = 32;

        const posCount = this.series_label.length;
        const axisYMax = this.forceMaxYFlag ? this.forceMaxYValue : this.maxOfArrayArray(this.series_data);
        const axisYLabels = this.createAxisYLabels(axisYMax);
        const axisXLabels = this.createAxisXLabels(posCount);
        const seriesCount = this.series_data.length;
        const valuesBelow = new Array(posCount).fill(0.0);

        const mapperX = new this.Mapper(0, posCount - 1, 64, this.width - 20);
        const mapperY = new this.Mapper(0, axisYMax, 10 + yLegendSize, this.height - 64, true);

        // Save mappers for hit testing
        this.mapperX = mapperX;
        this.mapperY = mapperY;

        // Draw "y" (value axis) lines
        for (let value of axisYLabels) {
            const x1 = mapperX(0);
            const x2 = mapperX(posCount - 1);
            const y = mapperY(value);
            svgDraw.line(x1, y, x2, y, "#80808030", "2px");
        }

        // Draw series bottom up, one by one, remembering edge (stroke) data
        const stroke_save_by_series = [];
        for (let series = seriesCount - 1; series >= 0; --series) {
            stroke_save_by_series.push([])
        }

        for (let series = seriesCount - 1; series >= 0; --series) {
            const series_data = this.series_data[series];
            const stroke_save_list = stroke_save_by_series[series];

            // Draw spans that have data, one by one
            for (let nextSpanStart = 0; nextSpanStart < posCount;) {
                const [s, e] = this.getNextValueSpan(series_data, nextSpanStart);
                if (s >= posCount) {
                    break
                }

                if (s + 1 === e) {
                    // Single element, draw circles not lines
                    const i = s;
                    const x = mapperX(i);
                    const y = mapperY(series_data[i] + valuesBelow[i]);

                    svgDraw.circle(x, y, 5.0, this.data_options[series].fill)

                    const stroke_save = [];
                    stroke_save_list.push(stroke_save);
                    stroke_save.push({x: x - 7, y: y});
                    stroke_save.push({x: x + 7, y: y});
                } else {
                    const points = [];
                    for (let i = s; i < e; ++i) {
                        const x = mapperX(i);
                        const y = mapperY(series_data[i] + valuesBelow[i]);

                        if (i === s) {
                            points.push({x: x, y: mapperY(valuesBelow[i])});
                        }

                        points.push({x: x, y: y});

                        if (i === e - 1) {
                            points.push({x: x, y: mapperY(valuesBelow[i])});

                            if (series !== seriesCount - 1) {
                                for (let j = e - 2; j >= s; --j) {
                                    points.push({x: mapperX(j), y: mapperY(valuesBelow[j])});
                                }
                            }
                        }
                    }

                    const stroke_save = [];
                    stroke_save_list.push(stroke_save);

                    for (let i = s; i < e; ++i) {
                        const x = mapperX(i);
                        const y = mapperY(series_data[i] + valuesBelow[i]);

                        stroke_save.push({x: x, y: y})
                    }

                    svgDraw.polygon(points, this.data_options[series].fill)
                }

                for (let i = s; i < e; ++i) {
                    valuesBelow[i] += series_data[i]
                }

                nextSpanStart = e
            }
        }

        // Draw stroke at top of each filled area, separately using saved data
        for (let series = seriesCount - 1; series >= 0; --series) {
            const stroke_save_list = stroke_save_by_series[series];
            for (let stroke_save of stroke_save_list) {
                svgDraw.polyline(stroke_save, this.data_options[series].line);
            }
        }

        // Draw "y" (value axis) labels
        for (let raw of axisYLabels) {
            const rounded = this.roundAxisYValue(raw)
            const value =
                (raw != null && this.valueFormatter != null)
                    ? this.valueFormatter(rounded)
                    : '' + rounded;

            svgDraw.textRight(60, mapperY(raw) + 4, "legend-tick-label-" + textClassSuffix,
                value, this.chart_options.tick)
        }

        // Draw "x" (time axis) labels
        const formatter = new this.LabelFormatter(this.series_label);
        for (let i in axisXLabels) {
            const value = axisXLabels[i];

            if (i != 0 && i != axisXLabels.length - 1) {
                const tx = mapperX(value);
                const ty = mapperY(0.0);
                svgDraw.line(tx, ty - 4, tx, ty + 4, this.chart_options.line);
            }

            const v = formatter(value);
            const x = mapperX(value) + 10;
            const y = mapperY(0.0) + (tickFontSize >= 11 ? 26 : 20);

            svgDraw.beginTransform(x, y, - 30);

            svgDraw.textRight(0, 0, "legend-tick-label-" + textClassSuffix,
                v, this.chart_options.tick)

            svgDraw.endTransform()
        }
    }

    getTickFontSize() {
        if (this.width < 500) {
            return 9;
        }
        return 11;
    }

    getTitleFontSize() {
        if (this.width < 500) {
            return 13;
        }
        return 15;
    }

    createAxisYLabels(max) {
        const list = [];
        let prev = -1;
        for (let i = 0; i <= 4; ++i) {
            const curr = max * i / 4;
            if (prev !== curr) {
                prev = curr;
                list.push(curr)
            }
        }
        return list
    }

    roundAxisYValue(value) {
        if (value >= 10) {
            return Math.round(value)
        }
        return Math.round((value * 100) / 100)
    }

    createAxisXLabels(len) {
        let step;
        if (len === 30 + 1) {
            step = 5
        } else if (len === 36 + 1) {
            step = 6
        } else {
            step = 4
        }
        var start = 0;
        for (let i = len - 1; i >= 0; i -= step) {
            start = i
        }
        const list = [];
        for (let i = start; i < len; i += step) {
            list.push(i)
        }
        return list
    }

    getNextValueSpan(list, start) {
        const len = list.length;
        for (let s = start; s < len; ++s) {
            if (list[s] != null) {
                for (let e = s + 1; e < len; ++e) {
                    if (list[e] == null) {
                        return [s, e]
                    }
                }
                return [s, len]
            }
        }

        return [len, len]
    }

    roundValue(value) {
        if (value == null) {
            return "n/a"
        } else if (value >= 1000) {
            return Math.round(value)
        } else if (value >= 100) {
            return Math.round(value * 10.0) / 10.0
        } else {
            return Math.round(value * 100.0) / 100.0
        }
    }

    maxOfArray(list) {
        if (list.length === 0) {
            return NaN
        }
        let t = null;
        for (let v of list) {
            if (v != null) {
                if (t == null || t < v) {
                    t = v
                }
            }
        }

        return this.roundMaxOfArray(t)
    }

    maxOfArrayArray(list) {
        let getValue = null;
        switch (list.length) {
            case 1:
                getValue = (i) => {
                    return list[0][i]
                };
                break;
            case 2:
                getValue = (i) => {
                    return list[0][i] + list[1][i]
                };
                break;
            case 3:
                getValue = (i) => {
                    return list[0][i] + list[1][i] + list[2][i]
                };
                break;
            case 4:
                getValue = (i) => {
                    return list[0][i] + list[1][i] + list[2][i] + list[3][i]
                };
                break;
            default:
                getValue = (i) => {
                    let v = 0.0;
                    for (let j = 0; j < list.length; ++j) {
                        v += list[j][i]
                    }
                    return v
                };
                break;
        }

        let t = null;
        for (let i in list[0]) {
            const v = getValue(i);
            if (v != null) {
                if (t == null || t < v) {
                    t = v
                }
            }
        }

        return this.roundMaxOfArray(t)
    }

    roundMaxOfArray(t) {
        if (t != null) {
            if (t <= 8) {
                return (Math.ceil(t * 10) + 1) / 10
            } else if (t < 20) {
                return Math.ceil(t) + 1
            } else if (t < 80) {
                return Math.ceil(t / 10 + 1) * 10
            }
        }
        return t
    }
}

