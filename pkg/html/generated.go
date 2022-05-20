// Code generated by pkg/html/html.go. DO NOT EDIT.
package html

const style = `
#topbar {
	padding: 5px 10px;
	background: #E0EBF5;
}

#topbar a {
	color: #375EAB;
	text-decoration: none;
}

h1, h2, h3, h4 {
	margin: 0;
	padding: 0;
	color: #375EAB;
	font-weight: bold;
}

.navigation_tab {
	border: 1px solid black;
	padding: 4px;
	margin: 4px;
}

.navigation_tab_selected {
	font-weight: bold;
	border: 2px solid black;
	padding: 4px;
	margin: 4px;
}

.position_table .navigation {
	padding-top: 15px;
	padding-bottom: 6px;
}

table {
	border: 1px solid #ccc;
	margin: 20px 5px;
	border-collapse: collapse;
	white-space: nowrap;
	text-overflow: ellipsis;
	overflow: hidden;
}

table caption {
	font-weight: bold;
}

table td, table th {
	vertical-align: top;
	padding: 2px 8px;
	text-overflow: ellipsis;
	overflow: hidden;
}

.namespace {
	font-weight: bold;
	font-size: large;
	color: #375EAB;
}

.position_table {
	border: 0px;
	margin: 0px;
	width: 100%;
	border-collapse: collapse;
}

.position_table td, .position_table tr {
	vertical-align: center;
	padding: 0px;
}

.position_table .namespace_td {
	width: 100%;
	padding-top: 10px;
	padding-left: 20px;
}

.position_table .search {
	text-align: right;
}

.list_table td, .list_table th {
	border-left: 1px solid #ccc;
}

.list_table th {
	background: #F4F4F4;
}

.list_table tr:nth-child(2n) {
	background: #F4F4F4;
}

.list_table tr:hover {
	background: #ffff99;
}

.list_table .namespace {
	width: 100pt;
	max-width: 100pt;
}

.list_table .title {
	width: 350pt;
	max-width: 350pt;
}

.list_table .commit_list {
	width: 500pt;
	max-width: 500pt;
}

.list_table .tag {
	font-family: monospace;
	font-size: 8pt;
	max-width: 60pt;
}

.list_table .opts {
	width: 40pt;
	max-width: 40pt;
}

.list_table .status {
	width: 250pt;
	max-width: 250pt;
}

.list_table .patched {
	width: 60pt;
	max-width: 60pt;
	text-align: center;
}

.list_table .kernel {
	width: 80pt;
	max-width: 80pt;
}

.list_table .maintainers {
	width: 150pt;
	max-width: 150pt;
}

.list_table .result {
	width: 60pt;
	max-width: 60pt;
}

.list_table .stat {
	width: 55pt;
	font-family: monospace;
	text-align: right;
}

.list_table .bisect_status {
	width: 75pt;
	max-width: 75pt;
	font-family: monospace;
	text-align: right;
}

.list_table .date {
	width: 60pt;
	max-width: 60pt;
	font-family: monospace;
	text-align: right;
}

.list_table .stat_name {
	width: 150pt;
	max-width: 150pt;
	font-family: monospace;
}

.list_table .stat_value {
	width: 120pt;
	max-width: 120pt;
	font-family: monospace;
}

.bad {
	color: #f00;
	font-weight: bold;
}

.inactive {
	color: #888;
}

.plain {
	text-decoration: none;
}

textarea {
	width:100%;
	font-family: monospace;
}

.mono {
	font-family: monospace;
}

.info_link {
	color: #25a7db;
	text-decoration: none;
}

.page {
	position: relative;
	width: 100%;
}

aside {
	position: absolute;
	top: 0;
	left: 0;
	bottom: 0;
	width: 290px;
	margin-top: 5px;
}

.panel {
	border: 1px solid #aaa;
	border-radius: 5px;
	margin-bottom: 5px;
	margin-top: 5px;
}

.panel h1 {
	font-size: 16px;
	margin: 0;
	padding: 2px 8px;
}

.panel select {
	padding: 5px;
	border: 0;
	width: 100%;
}

.panel label {
	margin-left: 7px;
}

.main-content {
	position: absolute;
	top: 0;
	left: 300px;
	right: 5px;
	min-height: 200px;
	overflow: hidden;
}

.graph_help {
	position: absolute;
	top: 115px;
	left: 10px;
	z-index: 1;
	text-decoration: none;
	font-weight: bold;
	font-size: xx-large;
	color: blue;
}

#graph_div {
	height: 85vh;
}

#crash_div {
	align: left;
	width: 90%;
	height: 400px;
	margin: 0 0;
	overflow: scroll;
	border: 1px solid #777;
	padding: 0px;
	background: transparent;
}

.input-values {
	margin-left: 7px;
	margin-bottom: 7px;
}

.input-group {
	margin-top: 7px;
	margin-bottom: 7px;
	display: block;
}

.input-group button {
	width: 20pt;
}
`
const js = `
// Copyright 2018 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

function sortTable(item, colName, conv, desc = false) {
	table = item.parentNode.parentNode.parentNode.parentNode;
	rows = table.rows;
	col = findColumnByName(rows[0].getElementsByTagName("th"), colName);
	values = [];
	for (i = 1; i < rows.length; i++)
		values.push([conv(rows[i].getElementsByTagName("td")[col].textContent), rows[i]]);
	if (desc)
		desc = !isSorted(values.slice().reverse())
	else
		desc = isSorted(values);
	values.sort(function(a, b) {
		if (a[0] == b[0]) return 0;
		if (desc && a[0] > b[0] || !desc && a[0] < b[0]) return -1;
		return 1;
	});
	for (i = 0; i < values.length; i++)
		table.tBodies[0].appendChild(values[i][1]);
	return false;
}

function findColumnByName(headers, colName) {
	for (i = 0; i < headers.length; i++) {
		if (headers[i].textContent == colName)
			return i;
	}
	return 0;
}

function isSorted(values) {
	for (i = 0; i < values.length - 1; i++) {
		if (values[i][0] > values[i + 1][0])
			return false;
	}
	return true;
}

function textSort(v) { return v == "" ? "zzz" : v.toLowerCase(); }
function numSort(v) { return -parseInt(v); }
function floatSort(v) { return -parseFloat(v); }
function reproSort(v) { return v == "C" ? 0 : v == "syz" ? 1 : 2; }
function patchedSort(v) { return v == "" ? -1 : parseInt(v); }
function lineSort(v) { return -v.split(/\r\n|\r|\n/g).length }

function timeSort(v) {
	if (v == "now")
		return 0;
	m = v.indexOf('m');
	h = v.indexOf('h');
	d = v.indexOf('d');
	if (m > 0 && h < 0)
		return parseInt(v);
	if (h > 0 && m > 0)
		return parseInt(v) * 60 + parseInt(v.substring(h + 1));
	if (d > 0 && h > 0)
		return parseInt(v) * 60 * 24 + parseInt(v.substring(d + 1)) * 60;
	if (d > 0)
		return parseInt(v) * 60 * 24;
	return 1000000000;
}



function findAncestorByClass (el, cls) {
	while ((el = el.parentElement) && !el.classList.contains(cls));
	return el;
}

function deleteInputGroup(node) {
	group = findAncestorByClass(node, "input-group")
	values = findAncestorByClass(group, "input-values")
	if (!values) {
		return false
	}
	count = values.querySelectorAll('.input-group').length
	if (count == 1) {
		// If it's the only input, just clear it.
		input = group.querySelector('input')
		input.value = ""
	} else {
		group.remove()
	}
	return false
}

function addInputGroup(node) {
	values = findAncestorByClass(node, "input-values")
	groups = values.querySelectorAll(".input-group")
	if (groups.length == 0) {
		// Something strange has happened.
		return false
	}
	lastGroup = groups[groups.length - 1]
	newGroup = lastGroup.cloneNode(true)
	newGroup.querySelector('input').value = ""
	values.insertBefore(newGroup, lastGroup.nextSibling)
	return false
}
`
