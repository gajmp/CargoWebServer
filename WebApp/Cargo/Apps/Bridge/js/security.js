/**
 * Role and permission are use to manage access to ressources.
 */
var RolePermissionManager = function (parent) {

    // Keep reference to the parent.
    this.parent = parent

    // The interface panel.
    this.panel = parent.appendElement({ "tag": "div", "class": "role_permission_manager" }).down()

    // So here I will create a tab panel...
    this.content = this.panel.appendElement({ "tag": "div", "style": "display:table;" }).down()
        .appendElement({ "tag": "div", "id": "role_tab", "class": "file_tab active", "style": "display: table-cell;", "innerHtml": "Roles" })
        .appendElement({ "tag": "div", "id": "permission_tab", "class": "file_tab", "style": "display: table-cell;", "innerHtml": "Permissions" })
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%;" })
        .appendElement({ "tag": "div", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "id": "append_item_button", "class": "fa fa-plus" }).up().up()
        .appendElement({ "tag": "div", "class":"security_manager_content"}).down()

    this.roleTab = this.panel.getChildById("role_tab")
    this.permissionTab = this.panel.getChildById("permission_tab")
    this.appendItemBtn = this.panel.getChildById("append_item_button")

    var appendRole = function (rolePermissionManager) {
        return function () {
            console.log("append role")
            // In that case I will display a dialog and ask the user
            // for the new role name to create.
            var dialog = new Dialog("new_role_dialog", rolePermissionManager.panel, true, "New Role")

            var input = dialog.content
                .appendElement({ "tag": "span", "style": "display: table-cell", "innerHtml": "Enter the role name to create: " })
                .appendElement({ "tag": "div", "style": "display: table-cell; vertical-align: middle; padding-right: 10px;" }).down()
                .appendElement({ "tag": "input", "type": "text" }).down()

            // Set the dialog to center of the screen.
            dialog.setCentered()

            // Fit the width to the content.
            dialog.fitWidthToContent()

            // Set the focus to the input box.
            input.element.focus()

            // Action here.
            dialog.okCallback = function (input) {
                return function () {
                    server.securityManager.createRole(input.element.value,
                        // successCallback
                        function (results, caller) {
                            // Nothing to do here... use event instead!
                        },
                        // errorCallback
                        function (errObj, caller) {
                            console.log(errObj)
                        }, {})
                }
            } (input)
        }
    } (this)

    var appendPermission = function () {
        console.log("append permission")
    }

    // by default onclick...
    this.appendItemBtn.element.onclick = appendRole

    // The roles...
    this.roleManager = new RoleManager(this.content)

    // The permissions...
    this.permissionManager = new PermissionManager(this.content)

    // Connect the action's
    this.roleTab.element.onclick = function (permissionTab, roleManager, permissionManager, appendItemBtn, appendRole) {
        return function () {
            permissionTab.element.className = "file_tab"
            this.className = "file_tab active"
            roleManager.panel.element.style.display = ""
            permissionManager.panel.element.style.display = "none"
            appendItemBtn.element.style.display = ""
            appendItemBtn.element.onclick = appendRole
        }
    } (this.permissionTab, this.roleManager, this.permissionManager, this.appendItemBtn, appendRole)

    this.permissionTab.element.onclick = function (roleTab, roleManager, permissionManager, appendItemBtn, appendPermission) {
        return function () {
            roleTab.element.className = "file_tab"
            this.className = "file_tab active"
            roleManager.panel.element.style.display = "none"
            permissionManager.panel.element.style.display = ""
            appendItemBtn.element.style.display = "none"
            appendItemBtn.element.onclick = appendPermission
        }
    } (this.roleTab, this.roleManager, this.permissionManager, this.appendItemBtn, appendPermission)

    // Connect the listener's
    // The new entity event listener
    server.entityManager.attach(this, NewEntityEvent, function (evt, rolePermissionManager) {
        if (evt.dataMap["entity"] != undefined) {
            var entity = entities[evt.dataMap["entity"].UUID]
            if (entity != undefined) {
                if (entity.TYPENAME == "CargoEntities.Role") {
                    rolePermissionManager.roleManager.displayRole(entity)
                } else if (entity.TYPENAME == "CargoEntities.Permission") {

                }
            }
        }
    })

    // The entity delete event listener
    server.entityManager.attach(this, DeleteEntityEvent, function (evt, rolePermissionManager) {
        // So here I will remove the line from the table...
        var entity = evt.dataMap["entity"]
        if (entity.TYPENAME == "CargoEntities.Role") {
            rolePermissionManager.roleManager.removeRole(entity)
        } else if (entity.TYPENAME == "CargoEntities.Permission") {

        }
    })

    return this
}

/**
 * The role manager is use to create, modify or delete roles.
 */
var RoleManager = function (parent) {
    // Keep reference to the parent.
    this.parent = parent

    // The interface panel.
    this.panel = parent.appendElement({ "tag": "div", "class": "role_manager", }).down()

    // Keep the roles in map.
    this.roles = {}

    // So here I will get the list of role from the server.
    server.entityManager.getEntities("CargoEntities.Role", "CargoEntities", null, 0, -1, [], false, false,
        // progressCallback
        function (index, total, caller) { },
        // successCallback
        function (results, caller) {
            var results = _.sortBy(results, 'M_id');
            for (var i = 0; i < results.length; i++) {
                caller.displayRole(results[i])
                caller.roles[results[i].UUID] = results[i]
            }
        },
        // errorCallback
        function (errObj, caller) {

        },
        this)

    /**
     * The update event.
     */
    server.entityManager.attach(this, UpdateEntityEvent, function (evt, roleManager) {
        if (evt.dataMap["entity"] != undefined) {
            var entity = entities[evt.dataMap["entity"].UUID]
            if (entity != undefined) {
                if (entity.TYPENAME == "CargoEntities.Role") {
                    // Romove the role if it already exist...
                    if (roleManager.roles[entity.UUID] != undefined) {
                        delete roleManager.roles[entity.UUID]
                    }
                    // display the role.
                    roleManager.displayRole(entity)
                } else if (entity.TYPENAME == "CargoEntities.Permission") {

                }
            }
        }
    })

    // Return the role manager.
    return this
}

/**
 * Display a new role.
 */
RoleManager.prototype.displayRole = function (role) {

    if (this.roles[role.UUID] != undefined) {
        return
    }

    this.roles[role.UUID] = role
    var isExpanded = false

    var row = this.panel.getChildById(role.M_id + "_table_row")
    if (row == undefined) {
        row = this.panel.appendElement({ "tag": "div", "id": role.M_id + "_table_row", "style": "display: table-row;" }).down()
    } else {
        isExpanded = this.panel.getChildById("collapse_" + role.M_id + "_btn").element.style.display == "table-cell"
        row.element.innerHTML = ""
    }

    row.appendElement({ "tag": "div", "id": role.M_id + "_header", "style": "display: table; width: 100%;" }).down()
        .appendElement({ "tag": "div", "id": "expand_" + role.M_id + "_btn", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-caret-right" }).up()
        .appendElement({ "tag": "div", "id": "collapse_" + role.M_id + "_btn", "class": "row_button", "style": "display: none " }).down() // switch to table-cell display...
        .appendElement({ "tag": "i", "class": "fa fa-caret-down" }).up()
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%;", "innerHtml": role.M_id })
        .appendElement({ "tag": "div", "id": "delete_" + role.M_id + "_btn", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-trash" })

    // Now the role panel...
    this.rolePanel = this.panel.getChildById(role.M_id + "_table_row").appendElement({ "tag": "div", "id": role.M_id + "_role_panel","class":"role_panel" }).down()

    // Here I will create tow colunm, the first will contain the accounts
    this.accountsCol = this.rolePanel.appendElement({ "tag": "div", "class":"role_table" }).down()

    this.accountsDiv = this.accountsCol.appendElement({ "tag": "div", "id": role.M_id + "account_table_header", "class":"role_table_header" }).down()
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%;", "innerHtml": "Accounts" })
        .appendElement({ "tag": "div", "id": "add_account_" + role.M_id + "_btn", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-plus" }).up().up()
        .appendElement({ "tag": "div", "id": role.M_id + "_accounts_div", "style": "display: block; width: 100%;" }).down()


    // Now i will append the accounts list.
    for (var i = 0; i < role.M_accountsRef.length; i++) {
        role["set_M_accountsRef_" + role.M_accountsRef[i] + "_ref"](
            function (roleManager, accountsDiv, role) {
                return function (ref) {
                    // i will append the account reference.
                    var row = accountsDiv.appendElement({ "tag": "div", "id": ref.M_id + "_table", "class":"role_table_row", "style": "display: table-row; margin-left: 24px;  width: 100%;" }).down()
                    row.appendElement({ "tag": "div", "id": "accountId", "class":"role_table_cell", "innerHtml": ref.M_name })
                        .appendElement({ "tag": "div", "id": "delete_" + ref.M_id + "_btn", "class": "row_button", "style": "display: table-cell; padding-left: 4px;" }).down()
                        .appendElement({ "tag": "i", "class": "fa fa-trash" })

                    var accountNameDiv = row.getChildById("accountId")
                    if (ref["set_M_userRef_" + ref.M_userRef + "_ref"] != undefined) {
                        ref["set_M_userRef_" + ref.M_userRef + "_ref"](
                            function (row) {
                                return function (ref) {
                                    var name = ref.M_firstName
                                    if (ref.M_middle.length > 0) {
                                        name += " " + ref.M_middle + " "
                                    }
                                    if (ref.M_lastName.indexOf("#") != 0) {
                                        name += " " + ref.M_lastName
                                    } else {
                                        name = ref.M_lastName.substring(ref.M_lastName.indexOf(" ") + 1)
                                    }
                                    accountNameDiv.element.title = name
                                }
                            } (row))
                    }

                    var deleteBtn = accountsDiv.getChildById("delete_" + ref.M_id + "_btn")
                    // Now the delete account action button.
                    deleteBtn.element.onclick = function (role, account) {
                        return function () {
                            console.log("role " + role.M_id + " delete account " + account.M_id)
                            server.securityManager.removeAccount(role.M_id, account.M_id,
                                // success callback
                                function () {
                                },
                                // error callback
                                function () { },
                                {})
                        }
                    } (role, ref)
                }
            } (this, this.accountsDiv, role)
        )
    }

    this.actionsCol = this.rolePanel.appendElement({ "tag": "div", "class":"role_table"  }).down()

    this.actionsDiv = this.actionsCol.appendElement({ "tag": "div", "id": role.M_id + "_actions_table_header", "class":"role_table_header" }).down()
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%;", "innerHtml": "Actions" })
        .appendElement({ "tag": "div", "id": "add_action_" + role.M_id + "_btn", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-plus" }).up().up()
        .appendElement({ "tag": "div", "id": role.M_id + "_actions_div", "style": "display: table; width: 100%;" }).down()

    // Now i will append the actions list.
    for (var i = 0; i < role.M_actionsRef.length; i++) {
        role["set_M_actionsRef_" + role.M_actionsRef[i] + "_ref"](
            function (roleManager, actionsDiv, role) {
                return function (ref) {
                    // i will append the account reference.
                    actionsDiv.appendElement({ "tag": "div", "id": ref.M_id + "_table", "class":"role_table_row" }).down()
                        .appendElement({ "tag": "div", "class":"role_table_cell", "innerHtml": ref.M_name })
                        .appendElement({ "tag": "div", "id": "delete_" + ref.M_id + "_btn", "class": "row_button", "style": "display: table-cell; padding-left: 4px;" }).down()
                        .appendElement({ "tag": "i", "class": "fa fa-trash" })

                    // Set the delete button.
                    var deleteBtn = actionsDiv.getChildById("delete_" + ref.M_id + "_btn")
                    if (role.M_id == "adminRole" || role.M_id == "guestRole") {
                        deleteBtn.element.style.display = "none"
                    }
                    // Now the delete action button.
                    deleteBtn.element.onclick = function (role, action) {
                        return function () {
                            console.log("role " + role.M_id + " delete action " + action.M_name)
                            server.securityManager.removeAction(role.M_id, action.M_name,
                                // success callback
                                function () {
                                },
                                // error callback
                                function () { },
                                {})
                        }
                    } (role, ref)
                }
            } (this, this.actionsDiv, role)
        )
    }

    // get buttons reference's
    this.expandBtn = this.panel.getChildById("expand_" + role.M_id + "_btn")
    this.collapseBtn = this.panel.getChildById("collapse_" + role.M_id + "_btn")
    this.deleteBtn = this.panel.getChildById("delete_" + role.M_id + "_btn")

    // append new account to role button.
    this.appendAccountBtn = this.panel.getChildById("add_account_" + role.M_id + "_btn")

    // Never delete guest or admin role.
    if (role.M_id == "adminRole" || role.M_id == "guestRole") {
        this.deleteBtn.element.style.display = "none"
    }

    // Set buttons action's
    this.appendAccountBtn.element.onclick = function (roleManager, role) {
        return function () {
            // Here I will get the list of all account...
            server.entityManager.getEntities("CargoEntities.Account", "CargoEntities", null, 0, -1, [], false, false,
                // Progress callback
                function (index, total, caller) { },
                // Success callback
                function (results, caller) {

                    var dialog = new Dialog("new_role_dialog", roleManager.panel, true, "Append account to " + role.M_id)

                    // first of all i will set the dialog with to be able to show more values.
                    dialog.content.element.style.display = "block"
                    dialog.content.element.style.height = "350px"
                    dialog.content.element.style.overflowY = "auto"
                    dialog.div.element.style.maxWidth = "300px"
                    // Keep the account to append to the role.
                    dialog.toAppend = {}

                    var results = _.sortBy(results, 'M_id');
                    for (var i = 0; i < results.length; i++) {
                        var account = results[i]

                        // Here I will remove account that already there...
                        if(caller.role.M_accounts == undefined){
                            caller.role.M_accounts = []
                        }
                        
                        if (!objectPropInArray(caller.role.M_accounts, "UUID", account.UUID)) {
                            var row = dialog.content.appendElement({ "tag": "div", "style": "display: table-row" }).down()
                            row.appendElement({ "tag": "div", "style": "display: table-cell; padding-left: 4px;" }).down()
                                .appendElement({ "tag": "input", "id": account.UUID + "_input", "type": "checkbox" }).up()
                                .appendElement({ "tag": "div", "style": "display: table-cell;padding-left: 4px;", "innerHtml": account.M_name })

                            if (account.M_userRef != undefined) {
                                if (account["set_M_userRef_" + account.M_userRef + "_ref"] != undefined) {
                                    // Call set_M_userRef_xxx_ref() function
                                    account["set_M_userRef_" + account.M_userRef + "_ref"](
                                        function (row) {
                                            return function (ref) {
                                                var name = ref.M_firstName

                                                if (ref.M_middle.length > 0) {
                                                    name += " " + ref.M_middle + " "
                                                }
                                                if (ref.M_lastName.indexOf("#") != 0) {
                                                    name += " " + ref.M_lastName
                                                } else {
                                                    name = ref.M_lastName.substring(ref.M_lastName.indexOf(" ") + 1)
                                                }

                                                row.appendElement({ "tag": "div", "style": "display: table-cell; text-align: left; padding-left: 15px;", "innerHtml": name })
                                            }
                                        } (row)
                                    )
                                }
                            }

                            var input = row.getChildById(account.UUID + "_input")
                            input.element.onclick = function (account, dialog) {
                                return function () {
                                    if (this.checked) {
                                        // append to the map.
                                        dialog.toAppend[account.UUID] = account
                                    } else {
                                        // remove it from the map.
                                        delete dialog.toAppend[account.UUID]
                                    }
                                }
                            } (account, dialog)

                            // Now the ok button.
                            dialog.okCallback = function (dialog, role) {
                                return function () {
                                    for (var id in dialog.toAppend) {
                                        var account = dialog.toAppend[id]
                                        // Here I will append the account to the role.
                                        server.securityManager.appendAccount(role.M_id, account.M_id,
                                            function (result, caller) { },
                                            function (errObj, caller) {
                                                console.log(errObj)
                                            }, {})
                                    }
                                }
                            } (dialog, role)
                        }
                    }

                    // Set the dialog to center of the screen.
                    dialog.setCentered()
                    dialog.fitWidthToContent()
                },
                // Error callback
                function (results, caller) {
                    console.log(errObj)
                },
                { "roleManager": roleManager, "role": role })
        }
    } (this, role)

    // Append action button
    this.appendActionBtn = this.panel.getChildById("add_action_" + role.M_id + "_btn")

    this.appendActionBtn.element.onclick = function (roleManager, role) {
        return function () {
            server.entityManager.getEntities("CargoEntities.Action", "CargoEntities", null, 0, -1, [], false, false,
                // Progress callback
                function (index, total, caller) { },
                // Success callback
                function (results, caller) {

                    var dialog = new Dialog("new_role_dialog", roleManager.panel, true, "Append account to " + role.M_id)

                    // first of all i will set the dialog with to be able to show more values.
                    dialog.content.element.style.display = "block"
                    dialog.content.element.style.height = "350px"
                    dialog.content.element.style.overflowY = "auto"
                    dialog.div.element.style.maxWidth = "350px"
                    // Keep the account to append to the role.
                    dialog.toAppend = {}

                    var results = _.sortBy(results, 'M_name');
                    for (var i = 0; i < results.length; i++) {
                        var action = results[i]
                        if (!objectPropInArray(caller.role.M_actions, "UUID", action.UUID)) {
                            var row = dialog.content.appendElement({ "tag": "div", "style": "display: table-row" }).down()
                            row.appendElement({ "tag": "div", "style": "display: table-cell; padding-left: 4px;" }).down()
                                .appendElement({ "tag": "input", "id": action.UUID + "_input", "type": "checkbox" }).up()
                                .appendElement({ "tag": "div", "style": "display: table-cell; padding-left: 4px; text-align: left;", "innerHtml": action.M_name })

                            var input = row.getChildById(action.UUID + "_input")
                            input.element.onclick = function (action, dialog) {
                                return function () {
                                    if (this.checked) {
                                        // append to the map.
                                        dialog.toAppend[action.UUID] = action
                                    } else {
                                        // remove it from the map.
                                        delete dialog.toAppend[action.UUID]
                                    }
                                }
                            } (action, dialog)
                        }
                    }

                    // Now the ok button.
                    dialog.okCallback = function (dialog, role) {
                        return function () {
                            for (var id in dialog.toAppend) {
                                var action = dialog.toAppend[id]
                                // Here I will append the action to the role.
                                server.securityManager.appendAction(role.M_id, action.M_name,
                                    function (result, caller) { },
                                    function (errObj, caller) {
                                        console.log(errObj)
                                    }, {})
                            }
                        }
                    } (dialog, role)

                    // Set the dialog to center of the screen.
                    dialog.setCentered()
                    dialog.fitWidthToContent()
                },
                // Error callback.
                function (errObj, caller) {
                    console.log(errObj)
                },
                { "roleManager": roleManager, "role": role })
        }
    } (this, role)

    this.expandBtn.element.onclick = function (collapseBtn, rolePanel) {
        return function () {
            this.style.display = "none"
            collapseBtn.element.style.display = "table-cell"
            rolePanel.element.style.display = "table"
        }
    } (this.collapseBtn, this.rolePanel)

    this.collapseBtn.element.onclick = function (expandBtn, rolePanel) {
        return function () {
            this.style.display = "none"
            expandBtn.element.style.display = "table-cell"
            rolePanel.element.style.display = "none"
        }
    } (this.expandBtn, this.rolePanel)

    this.deleteBtn.element.onclick = function (roleManager, role, row) {
        return function () {
            row.element.style.display = "none"
            // Now here i will remove the role from the panel.
            server.securityManager.deleteRole(role.M_id,
                // The success callback
                function (result, caller) {

                },
                // The error callback.
                function (errObj, caller) {

                }, {})
        }
    } (this, role, row)

    if (isExpanded) {
        this.expandBtn.element.click()
    }
}

/** 
 * Remove a given role from the list.
 */
RoleManager.prototype.removeRole = function (role) {
    if (this.roles[role.UUID] == undefined) {
        return
    }

    var roleRow = this.panel.getChildById(role.M_id + "_table_row")
    if (roleRow != undefined) {
        roleRow.parentElement.removeElement(roleRow)
    }

    // Clear the element.
    roleRow.element.innerHTML = ""

    delete this.roles[role.UUID]
}

///////////////////////////////////// Permission /////////////////////////////////////

/**
 * The permission manager is use to change permission of an entity.
 */
var PermissionManager = function (parent) {
    // Keep reference to the parent.
    this.parent = parent

    // The interface panel.
    this.panel = parent.appendElement({ "tag": "div", "class": "permission_manager", "style": "display:none;" }).down()

    // The search panel...
    this.panel.appendElement({ "tag": "div", "style": "display: table; width: 99%; padding: 5px;" }).down()
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%;" })
        .appendElement({ "tag": "input", "id": "search_permission_input", "type": "text", "style": "display: table-cell;", "placeholder":"Search" })
        .appendElement({ "tag": "div", "id": "search_permission_button", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-search" })

    // The results div.
    this.resultsDiv = this.panel.appendElement({ "tag": "div", "style": "display: table; width: 99%; padding: 5px;" }).down()

    // The search button.
    this.searchInput = this.panel.getChildById("search_permission_input")
    this.searchBtn = this.panel.getChildById("search_permission_button")

    this.searchBtn.element.onclick = function (permissionManager) {
        return function () {
            var values = permissionManager.searchInput.element.value.split(" ")
            var q = new EntityQuery("CargoEntities.User")

            if (values.length == 1) {
                q.Query = "CargoEntities.User.M_firstName ~=\"" + values[0] + "\" || CargoEntities.User.M_lastName ~=\"" + values[0] + "\""
            } else if (values.length == 2) {
                q.Query = "CargoEntities.User.M_firstName ~=\"" + values[0] + "\" && CargoEntities.User.M_lastName ~=\"" + values[1] + "\""
            }

            // Clear previous results.
            permissionManager.resultsDiv.removeAllChilds()
            permissionManager.resultsDiv.element.innerHTML = ""

            server.entityManager.getEntities("CargoEntities.User", "CargoEntities", q, 0, -1, [], false, false,
                // Progress...
                function () {

                },
                // Sucess...
                function (results, caller) {
                    console.log(results)
                    for (var i = 0; i < results.length; i++) {
                        var user = results[i]
                        // I will get the associated user account and display permission they contain.
                        for (var j = 0; j < user.M_accounts.length; j++) {
                            user["set_M_accounts_" + user.M_accounts[j] + "_ref"](
                                function (permissionManager, user) {
                                    return function (account) {
                                        permissionManager.displayPermissions(account, user)
                                    }
                                } (permissionManager, user)
                            )
                        }
                    }
                },
                function () {

                }, permissionManager)
        }
    } (this)

    // Return the permission manager.
    return this
}

/**
 * Display the permissions for a given account.
 */
PermissionManager.prototype.displayPermissions = function (account, user) {

    // Now display the new list.
    var row = this.resultsDiv.getChildById("permissions_" + user.M_id + "_row")

    if (row == undefined) {
        row = this.resultsDiv.appendElement({ "tag": "div", "id": "permissions_" + user.M_id + "_row", "style": "display: table-row;" }).down()
    } else {
        return
    }

    // The user name.
    var userName = user.M_firstName

    if (user.M_middle.length > 0) {
        userName += " " + user.M_middle + " "
    }

    userName += " " + user.M_lastName

    // The account name.
    row.appendElement({ "tag": "div", "id": user.M_id + "_header", "style": "display: table; width: 100%;" }).down()
        .appendElement({ "tag": "div", "id": "expand_" + user.M_id + "_btn", "class": "row_button", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-caret-right" }).up()
        .appendElement({ "tag": "div", "id": "collapse_" + user.M_id + "_btn", "class": "row_button", "style": "display: none " }).down() // switch to table-cell display...
        .appendElement({ "tag": "i", "class": "fa fa-caret-down" }).up()
        .appendElement({ "tag": "div", "style": "display: table-cell;width: 100%;", "innerHtml": userName + " (" + account.M_name + ")" })

    // The permission panel.
    var permissionPanel = row.appendElement({ "tag": "div", "class":"permissions_panel" }).down()
    // The header.
    permissionPanel.appendElement({ "tag": "div", "class":"permissions_panel_header" }).down()
        .appendElement({ "tag": "div", "style": "display: table-cell; width: 100%; padding: 3px 0px 2px 4px; vertical-align: middle;", "innerHtml": "Permissions" })
        .appendElement({ "tag": "div", "id": "new_permission_" + user.M_id + "_btn", "class": "row_button", "style": "display: table-cell; padding: 3px 0px 2px 4px;vertical-align: middle;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-plus" }).up()

    // Now the display the actual permission.
    for (var i = 0; i < account.M_permissionsRef.length; i++) {

        if (isString(account.M_permissionsRef[i])) {
            account["set_M_permissionsRef_" + account.M_permissionsRef[i] + "_ref"](
                function (permissionManager, permissionPanel, account) {
                    return function (ref) {
                        permissionManager.displayPermission(permissionPanel, ref, account)
                    }
                } (this, permissionPanel, account)

            )
        } else {
            this.displayPermission(permissionPanel, account.M_permissionsRef[i], account)
        }
    }

    // The permission expand and collapse button
    var expandBtn = this.panel.getChildById("expand_" + user.M_id + "_btn")
    var collapseBtn = this.panel.getChildById("collapse_" + user.M_id + "_btn")
    var appendPermission = this.panel.getChildById("new_permission_" + user.M_id + "_btn")

    expandBtn.element.onclick = function (collapseBtn, permissionPanel) {
        return function () {
            this.style.display = "none"
            collapseBtn.element.style.display = "table-cell"
            permissionPanel.element.style.display = "table"
        }
    } (collapseBtn, permissionPanel)

    collapseBtn.element.onclick = function (expandBtn, permissionPanel) {
        return function () {
            this.style.display = "none"
            expandBtn.element.style.display = "table-cell"
            permissionPanel.element.style.display = "none"
        }
    } (expandBtn, permissionPanel)

    // Append/Create permission and assign it to a user.
    appendPermission.element.onclick = function (permissionManager, account, permissionPanel) {
        return function () {
            // Here I will create the new permission.
            var permission = eval("new CargoEntities.Permission()")
            permission.M_types = 0
            permission.M_id = ""
            permissionManager.displayPermission(permissionPanel, permission, account)
        }
    } (this, account, permissionPanel)
}

/**
 * Display single permission panel.
 */
PermissionManager.prototype.displayPermission = function (parent, permission, account) {

    // does nothing if the panel is already there.
    if (parent.getChildById("permission_" + permission.M_id + "_panel") != undefined) {
        return
    }

    var panel = parent.appendElement({ "tag": "div","class":"permission_panel", "id": "permission_" + permission.M_id + "_panel"}).down()
    panel.appendElement({ "tag": "table", "style": "display: table; width: 100%;" }).down()
        .appendElement({ "tag": "div", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "label", "for": "permission_id_" + permission.UUID + "_input", "innerHtml": "Filter " })
        .appendElement({ "tag": "input", "id": "permission_id_" + permission.UUID + "_input", "type": "text", "style": "width: 75%;" }).up()
        .appendElement({ "tag": "div", "style": "display: table-cell;" }).down()
        .appendElement({ "tag": "div", "id": "save_" + permission.UUID + "_btn", "class": "row_button", "style": "display:none;" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-save" }).up()
        .appendElement({ "tag": "div", "id": "delete_" + permission.UUID + "_btn", "class": "row_button" }).down()
        .appendElement({ "tag": "i", "class": "fa fa-trash" }).up()

    // Now the permission type.
    panel.appendElement({ "tag": "table", "style": "display: table; width: 100%;" }).down()
        // Read button
        .appendElement({ "tag": "div", "style": "display: table-cell;width: 33.33%;" }).down()
        .appendElement({ "tag": "label", "for": "permission_read_" + permission.UUID + "_check", "innerHtml": "Read " })
        .appendElement({ "tag": "input", "type": "checkbox", "id": "permission_read_" + permission.UUID + "_check" }).up()
        // Write button
        .appendElement({ "tag": "div", "style": "display: table-cell;width: 33.33%;" }).down()
        .appendElement({ "tag": "label", "for": "permission_write_" + permission.UUID + "_check", "innerHtml": "Write " })
        .appendElement({ "tag": "input", "type": "checkbox", "id": "permission_write_" + permission.UUID + "_check" }).up()
        // Execute button.
        .appendElement({ "tag": "div", "style": "display: table-cell;width: 33.33%;" }).down()
        .appendElement({ "tag": "label", "for": "permission_exec_" + permission.UUID + "_check", "innerHtml": "Execute " })
        .appendElement({ "tag": "input", "type": "checkbox", "id": "permission_exec_" + permission.UUID + "_check" })

    // Save and delete button.
    var saveBtn = this.panel.getChildById("save_" + permission.UUID + "_btn")
    var deleteBtn = this.panel.getChildById("delete_" + permission.UUID + "_btn")
    var patternInput = this.panel.getChildById("permission_id_" + permission.UUID + "_input")
    // Now the permission.
    var readCheckbox = this.panel.getChildById("permission_read_" + permission.UUID + "_check")
    var writeCheckbox = this.panel.getChildById("permission_write_" + permission.UUID + "_check")
    var execCheckbox = this.panel.getChildById("permission_exec_" + permission.UUID + "_check")

    // Set the values...
    var setPermissionType = function (permission, readCheckbox, writeCheckbox, execCheckbox) {
        if (!readCheckbox.element.checked && !writeCheckbox.element.checked && !execCheckbox.element.checked) {
            permission.M_types = 0
        } else if (!readCheckbox.element.checked && !writeCheckbox.element.checked && execCheckbox.element.checked) {
            permission.M_types = 1
        } else if (!readCheckbox.element.checked && writeCheckbox.element.checked && !execCheckbox.element.checked) {
            permission.M_types = 2
        } else if (!readCheckbox.element.checked && writeCheckbox.element.checked && execCheckbox.element.checked) {
            permission.M_types = 3
        } else if (readCheckbox.element.checked && !writeCheckbox.element.checked && !execCheckbox.element.checked) {
            permission.M_types = 4
        } else if (readCheckbox.element.checked && !writeCheckbox.element.checked && execCheckbox.element.checked) {
            permission.M_types = 5
        } else if (readCheckbox.element.checked && writeCheckbox.element.checked && !execCheckbox.element.checked) {
            permission.M_types = 6
        } else if (readCheckbox.element.checked && writeCheckbox.element.checked && execCheckbox.element.checked) {
            permission.M_types = 7
        }
    }

    if (permission.UUID.length > 0) {
        patternInput.element.value = permission.M_id
        if (permission.M_types == 0) {
            readCheckbox.element.checked = false
            writeCheckbox.element.checked = false
            execCheckbox.element.checked = false
        } else if (permission.M_types == 1) {
            readCheckbox.element.checked = false
            writeCheckbox.element.checked = false
            execCheckbox.element.checked = true
        } else if (permission.M_types == 2) {
            readCheckbox.element.checked = false
            writeCheckbox.element.checked = true
            execCheckbox.element.checked = false
        } else if (permission.M_types == 3) {
            readCheckbox.element.checked = true
            writeCheckbox.element.checked = false
            execCheckbox.element.checked = false
        } else if (permission.M_types == 4) {
            readCheckbox.element.checked = true
            writeCheckbox.element.checked = false
            execCheckbox.element.checked = false
        } else if (permission.M_types == 5) {
            readCheckbox.element.checked = true
            writeCheckbox.element.checked = false
            execCheckbox.element.checked = true
        } else if (permission.M_types == 6) {
            readCheckbox.element.checked = true
            writeCheckbox.element.checked = true
            execCheckbox.element.checked = false
        } else if (permission.M_types == 7) {
            readCheckbox.element.checked = true
            writeCheckbox.element.checked = true
            execCheckbox.element.checked = true
        }
    }

    // Set the permission type.
    readCheckbox.element.onclick = writeCheckbox.element.onclick = execCheckbox.element.onclick = function (permission, readCheckbox, writeCheckbox, execCheckbox, saveBtn) {
        return function () {
            setPermissionType(permission, readCheckbox, writeCheckbox, execCheckbox)
            saveBtn.element.style.display = ""
        }
    } (permission, readCheckbox, writeCheckbox, execCheckbox, saveBtn)

    // The button action's.
    saveBtn.element.onclick = function (permission, account) {
        return function () {
            this.style.display = "none"
            // Save the permission.
            server.securityManager.appendPermission(account.M_id, permission.M_types, permission.M_id,
                // successCallback
                function (result, caller) {

                },
                // errorCallback
                function (errObj, caller) {
                    console.log(errObj)
                },
                {})
        }
    } (permission, account)

    deleteBtn.element.onclick = function (account, permission, panel) {
        return function () {
            if (permission.UUID.length > 0) {
                server.securityManager.removePermission(account.M_id, permission.M_id,
                    // Success callback
                    function (result, caller) {
                    },
                    // Error callback
                    function (errObj, caller) {

                    },
                    {})
            }
            // remove the panel from the view.
            panel.parentElement.removeElement(panel)
        }
    } (account, permission, panel)

    patternInput.element.onchange = function (saveBtn, permission, panel) {
        return function () {
            saveBtn.element.style.display = ""
            if (this.value.length == 0) {
                saveBtn.element.style.display = "none"
            } else {
                permission.M_id = this.value
                panel.element.id = "permission_" + permission.M_id + "_panel"
            }
        }
    } (saveBtn, permission, panel)

    console.log(permission)
}