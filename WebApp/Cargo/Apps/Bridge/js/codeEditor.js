/**
 * The code editor
 * TODO create the split functionnality
 * TODO create the multiuser access for a single file
 */

var CodeEditor = function (parent) {

    // The panel...
    this.panel = parent.appendElement({ "tag": "div", "class": "codeEditor" }).down()

    // The open files...
    this.files = {}

    // The toolbars associated whit each editor.
    this.toolbars = {}

    // The current file.
    this.activeFile = null

    // The editor
    this.editors = {}

    // The map of file panels.
    this.filesPanel = {}

    // TODO create the new file event and the delete file event here...
    this.quiet = false

    // Here I will create the file toolbar...
    //this.fileToolbar = new Element(null, { "tag": "div", "class": "toolbar" })
    this.theme = localStorage.getItem("bridge_editor_theme")
    if (this.theme == undefined) {
        this.theme = "ace/theme/chrome"
    }

    this.themeClass = localStorage.getItem("bridge_editor_theme_class")
    if (this.themeClass == undefined) {
        this.themeClass = "ace-chrome"
    }

    // Here I will attach the file navigator to file event.
    // Open.
    server.fileManager.attach(this, OpenEntityEvent, function (evt, codeEditor) {
        if (evt.dataMap["fileInfo"] !== undefined) {
            var file = entities[evt.dataMap["fileInfo"].UUID]
            if (file == undefined) {
                file = evt.dataMap["fileInfo"]
            }

            if (file.M_data !== undefined) {
                // Here thats mean the file was open
                codeEditor.appendFile(file)
            }

        } else if (evt.dataMap["bpmnDiagramInfo"] !== undefined) {
            var diagram = entities[evt.dataMap["bpmnDiagramInfo"].UUID]
            if (diagram !== undefined) {
                codeEditor.appendBpmnDiagram(diagram)
            }
        } else if (evt.dataMap["prototypeInfo"] != undefined) {
            var prototype = evt.dataMap["prototypeInfo"]
            if (prototype !== undefined) {
                codeEditor.appendPrototypeEditor(prototype)
            }
        }
    })

    // Attach the file close event.
    server.fileManager.attach(this, CloseEntityEvent, function (evt, codeEditor) {
        var fileId = evt.dataMap["fileId"]
        if (fileId !== undefined) {
            codeEditor.removeFile(fileId)
            if (codeEditor.toolbars[fileId] !== undefined) {
                for (var i = 0; i < codeEditor.toolbars[fileId].length; i++) {
                    var toolbar = codeEditor.toolbars[fileId][i];
                    homepage.toolbarDiv.removeElement(toolbar);
                }
            }
            codeEditor.toolbars[fileId] = []
        }
    })

    // Attach the file update event.
    server.fileManager.attach(this, UpdateFileEvent, function (evt, codeEditor) {
        if (evt.dataMap.fileInfo !== undefined) {
            var file = evt.dataMap["fileInfo"]
            var editor = codeEditor.editors[file.M_id + "_editor"]
            if (editor !== undefined) {
                // Supend the change event propagation
                codeEditor.quiet = true
                var position = editor.getCursorPosition()
                editor.setValue(decode64(file.M_data), -1)
                editor.clearSelection()
                editor.scrollToLine(position.row + 1, true, true, function () { });
                editor.gotoLine(position.row + 1, position.column)
                // Resume the chage event propagation.
                codeEditor.quiet = false
            }
        }
    })

    server.fileManager.attach(this, ChangeThemeEvent, function (evt, codeEditor) {
        codeEditor.theme = evt.dataMap.theme
        for (var editorUuid in codeEditor.editors) {
            if(codeEditor.editors[editorUuid].setTheme !== undefined){
                codeEditor.editors[editorUuid].setTheme(evt.dataMap.theme);
            }else if(codeEditor.editors[editorUuid].editor.setTheme !== undefined){
                codeEditor.editors[editorUuid].editor.setTheme(evt.dataMap.theme);
            }
        }
    })

    server.entityManager.attach(this, UpdateEntityEvent, function (evt, codeEditor) {
        if (evt.dataMap.entity !== undefined) {
            var file = evt.dataMap["entity"]
            var editor = codeEditor.editors[file.M_id + "_editor"]
            if (editor !== undefined && file.TYPENAME == "CargoEntities.File") {
                // Supend the change event propagation
                codeEditor.quiet = true
                var position = editor.getCursorPosition()
                editor.setValue(decode64(file.M_data), -1)
                editor.clearSelection()
                editor.scrollToLine(position.row + 1, true, true, function () { });
                editor.gotoLine(position.row + 1, position.column)
                // Resume the chage event propagation.
                codeEditor.quiet = false
            }
        }
    })

    /** Always display the license. */
    /*server.fileManager.getFileByPath("/LICENSE",
        // Success callback 
        function (result, caller) {
            result.M_mime = "text/plain"
            evt = { "code": OpenEntityEvent, "name": FileEvent, "dataMap": { "fileInfo": result } }
            server.eventHandler.broadcastLocalEvent(evt)
        },
        // Error callback
        function (errObj, caller) {

        }, {})*/

    return this
}

/**
 * Here I will display the prototype editor.
 */
CodeEditor.prototype.appendPrototypeEditor = function (prototype) {
    // Here I will set the prototype editor.
    if (this.files[prototype.TypeName] !== undefined) {
        // Set the tab active...
        this.setActiveFile(prototype.TypeName)
        return
    }

    server.configurationManager.getActiveConfigurations(
        function (results, caller) {
            var namespaces = []
            for (var i = 0; i < results.M_dataStoreConfigs.length; i++) {
                // Sql entities are not part of the heritage system.
                if(results.M_dataStoreConfigs[i].M_dataStoreType == 2 && results.M_dataStoreConfigs[i].M_id != "sql_info"){
                    namespaces.push(results.M_dataStoreConfigs[i].M_id)
                }
            }
            var codeEditor = caller.codeEditor
            var prototype = caller.prototype
            var entityEditor = new EntityPrototypeEditor(filePanel, namespaces, undefined, function (entityEditor) {
                entityEditor.typeNameInput.element.value = prototype.TypeName
                entityEditor.setCurrentPrototype(prototype)
                entityEditor.space.element.style.display = ""
            })
        },
        function (errObj, caller) {

        },
        { "codeEditor": this, "prototype": prototype })

    var filePanel = this.panel.appendElement({ "tag": "div", "class": "filePanel", "id": prototype.TypeName + "_editor" }).down()
    this.files[prototype.TypeName] = prototype
    this.filesPanel[prototype.TypeName] = filePanel
    this.setActiveFile(prototype.TypeName)
}

CodeEditor.prototype.appendBpmnDiagram = function (diagram) {
    // Here I will set the file
    if (this.files[diagram.M_id] !== undefined) {
        // Set the tab active...
        this.setActiveFile(diagram.M_id)
        this.diagram.canvas.initWorkspace()
        return
    }

    var filePanel = this.panel.appendElement({ "tag": "div", "class": "filePanel", "id": diagram.M_id + "_editor" }).down()


    this.diagram = new SvgDiagram(filePanel, diagram)

    this.diagram.init(function (codeEditor, diagram, filePanel) {
        return function () {
            codeEditor.diagram.drawDiagramElements()

            codeEditor.files[diagram.M_id] = diagram
            codeEditor.filesPanel[diagram.M_id] = filePanel
            codeEditor.setActiveFile(diagram.M_id)

            // Now the resize element...
            codeEditor.diagram.canvas.initWorkspace = function (workspace) {
                return function () {
                    if (workspace.lastChild === undefined) {
                        return
                    }
                    if (workspace.lastChild.lastChild !== undefined) {
                        for (var childId in workspace.childs) {
                            var child = workspace.childs[childId];
                            if (child.element.viewBox !== null) {
                                if (child.resize != undefined) {
                                    child.resize(workspace.element.offsetWidth, workspace.element.offsetHeight);
                                }
                            }
                        }
                    }
                }
            }(filePanel)

            window.addEventListener("resize", function (canvas) {
                return function () {
                    canvas.initWorkspace()
                }
            }(codeEditor.diagram.canvas))

            codeEditor.diagram.canvas.initWorkspace()
        }
    }(this, diagram, filePanel))
}

CodeEditor.prototype.appendFile = function (file) {

    var fileMode = ""
    if (file.M_mime == "application/javascript") {
        fileMode = "ace/mode/javascript"
    } else if (file.M_mime == "text/css") {
        fileMode = "ace/mode/css"
    } else if (file.M_mime == "text/html") {
        fileMode = "ace/mode/html"
    } else if (file.M_mime == "text/json") {
        fileMode = "ace/mode/json"
    } else if (file.M_mime == "text/plain") {
        fileMode = "ace/mode/text"
    }


    // Here I will set the file
    if (this.files[file.M_id] != undefined) {
        // Set the tab active...
        this.setActiveFile(file.M_id)
        return
    }

    // Here the new file tab must be created.
    this.files[file.M_id] = file

    //var deleteBtn = fileToolbar.appendElement({"tag":"div"}).down()

    if (fileMode.length == 0) {
        if (file.M_name.endsWith(".eql") || file.M_name.endsWith(".sql")) {
            // Here I will create a query editor insted of ace editor.
            var filePanel = this.panel.appendElement({ "tag": "div", "class": "filePanel", "id": file.M_id + "_editor" }).down()

            // The query editor.
            var queryEditor = new QueryEditor(filePanel, file, function (codeEditor, fileId) {
                return function (queryEditor) {
                    // I will append the list of dataStore that can be use to do query.
                    codeEditor.toolbars[fileId] = []
                    codeEditor.toolbars[fileId].push(queryEditor.queryToolBar)
                }
            }(this, file.M_id))

            // Init the query editor.
            queryEditor.init()

            this.editors[file.M_id + "_editor"] = queryEditor
            this.filesPanel[file.M_id] = filePanel
            this.setActiveFile(file.M_id)
        }
        return
    }


    // Now I will create the file editor.
    var filePanel = this.panel.appendElement({ "tag": "xmp", "class": "filePanel", "id": file.M_id + "_editor", "innerHtml": decode64(file.M_data) }).down()
    
    var observer = new MutationObserver(function (codeEditor) {
        return function (multiRecord) {
            var record = multiRecord.pop()
            var themeClass = record.target.classList[record.target.classList.length - 1]
            var isDark = record.target.className.indexOf("ace_dark") != -1
            if (themeClass != codeEditor.themeClass && themeClass != "ace-tm") {
                // Keep it in the local storage.
                localStorage.setItem("bridge_editor_theme_class", themeClass)
                localStorage.setItem("bridge_editor_theme", codeEditor.theme)
                codeEditor.themeClass = themeClass
                evt = { "code": ChangeThemeEvent, "name": FileEvent, "dataMap": { "theme": codeEditor.theme, "themeClass": codeEditor.themeClass, "isDark" : isDark} }
                server.eventHandler.broadcastLocalEvent(evt)
            }
        }
    }(this))

    observer.observe(filePanel.element, {
        attributes: true,
        attributeFilter: ['class'],
        childList: false,
        characterData: false
    })

    ace.require("ace/ext/language_tools");
    var editor = ace.edit(file.M_id + "_editor");
    ace.require('ace/ext/settings_menu').init(editor);
    editor.setTheme(this.theme);
    editor.getSession().setMode(fileMode);

    editor.setOptions({
        enableBasicAutocompletion: true,
        enableSnippets: true,
        enableLiveAutocompletion: true
    });

    this.editors[file.M_id + "_editor"] = editor

    // Editor command here.
    editor.commands.addCommands([{
        name: "showSettingsMenu",
        bindKey: { win: "Ctrl-q", mac: "Ctrl-q" },
        exec: function (codeEditor) {
            return function (editor) {
                editor.showSettingsMenu();
                var themeSelect = document.getElementById('setTheme');
                themeSelect.addEventListener("change", function () {
                    // Here I will throw a change theme event.
                    codeEditor.theme = this.value
                });
            }
        }(this),
        readOnly: true
    }]);

    // In case of file update...
    editor.getSession().on('change', function (fileId, fileUUID, codeEditor) {
        return function () {
            if (!codeEditor.quiet && entities[fileUUID] !== undefined) {
                var editor = codeEditor.editors[fileId + "_editor"]
                var evt = { "code": ChangeFileEvent, "name": FileEvent, "dataMap": { "fileId": fileId } }
                var file = entities[fileUUID]
                file.M_data = encode64(editor.getSession().getValue())
                server.eventHandler.broadcastLocalEvent(evt)
            }
        }
    }(file.M_id, file.UUID, this));

    this.filesPanel[file.M_id] = filePanel
    this.setActiveFile(file.M_id)
}

CodeEditor.prototype.removeFile = function (fileId) {
    if (this.filesPanel[fileId] != undefined) {
        // remove the element from the panel.
        this.panel.removeElement(this.filesPanel[fileId])
        delete this.filesPanel[fileId]
        delete this.files[fileId]
        delete this.editors[fileId + "_editor"]

        if (this.activeFile != undefined) {
            if (this.activeFile.M_id == fileId) {
                this.activeFile = null
            }
        }
    }
}

/**
 * Set the current file panel.
 */
CodeEditor.prototype.setActiveFile = function (fileId) {
    for (var id in this.filesPanel) {
        this.filesPanel[id].element.style.display = "none"
    }
    if (this.filesPanel[fileId] !== undefined) {
        this.filesPanel[fileId].element.style.display = ""
    }
    this.activeFile = this.files[fileId]

    // Now the toolbar...
    var toolbars = document.getElementsByClassName("toolbar")
    for (var i = 0; i < toolbars.length; i++) {
        toolbars[i].style.display = "none" // hide toolbar.
    }

    if (document.getElementById(fileId + "_toolbar") != undefined) {
        document.getElementById(fileId + "_toolbar").style.display = ""
    }
}