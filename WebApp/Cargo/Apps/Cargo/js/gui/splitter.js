
/**
 * Set splitter action...
 */
function initSplitter(splitter, area, min) {
	if (min == undefined) {
		min = 0;
	}
	
	var moveSplitter = function (splitter, area, min) {
		return function (e) {
			var isVertical = splitter.element.className.indexOf("vertical") > -1
			if (isVertical) {
				var splitterLeft = getCoords(splitter.element).left
				var areaLeft = getCoords(area.element).left
				var isAtLeft = splitterLeft >= areaLeft
				var offset = e.clientX - splitterLeft
				var w = area.element.offsetWidth
				if (isAtLeft) {
					if (w > min) {
						w = area.element.offsetWidth + offset
					}
				} else {
					w = area.element.offsetWidth - offset
				}
				area.element.style.width = w + "px"
			} else {
				var splitterTop = getCoords(splitter.element).top
				var areaTop = getCoords(area.element).top
				var isAtTop = splitterTop >= areaTop
				var offset = e.clientY - splitterTop
				var h = area.element.offsetHeight
				if (isAtTop) {
					h = area.element.offsetHeight + offset
				} else {
					h = area.element.offsetHeight - offset
				}
				area.element.style.height = h + "px"
			}
		}
	} (splitter, area, min)

	function mouseUp() {
		window.removeEventListener('mousemove', moveSplitter);
		window.removeEventListener('mouseup', mouseUp);
	}

	splitter.element.onmousedown = function () {
		window.addEventListener('mousemove', moveSplitter);
		window.addEventListener('mouseup', mouseUp);
	}

}