//
//
//
//
//
enum HttpMethods { GET = "GET", POST = "POST" };
enum AjaxError { ERROR, TIMEOUT, ABORT };
type AjaxSuccessCallback = (request: XMLHttpRequest, ev: Event) => void;
type AjaxErrorCallback = (request: XMLHttpRequest, ev: ProgressEvent<EventTarget>, error: AjaxError) => void;
const sleep = async (ms: number) => new Promise(r => setTimeout(r, ms));
let _interval_id: number;
let _debt_template: string;
let _users : Array<DatabaseUser>;
let _whoami: DatabaseUser;

type DatabaseUser = {
	ID: number,
	UPN: string
};

type PaymentRecord = {
	Payer: number,
	Payee: number,
	NumMeals: number
}

function _encode_cookie(uid: number) : string {
	return `uid=${uid}; samesite=strict`;
}

function _parse_cookie() : any {
	let obj: any = {};
	document.cookie.split(";").forEach((v) => {
		const [key, value] = v.split("=");
		obj[key] = value;
	})
	return obj;
}

function _sendRequest(method: HttpMethods = HttpMethods.GET,
		     endpoint: string,
		     onSuccess? : AjaxSuccessCallback,
		     onFailure? : AjaxErrorCallback,
		     body? : any)
{
	let start = performance.now();
	let ajax = new XMLHttpRequest();
	ajax.open(method, endpoint);

	ajax.setRequestHeader("Accept", "application/json");
	ajax.setRequestHeader("Content-Type", "application/json");
	if (onSuccess)
	{
		ajax.onreadystatechange = (ev) =>
		{
			if (ajax.readyState == ajax.DONE) 
			{
				onSuccess(ajax, ev);
				console.debug("request time: " + (performance.now() - start));
			}
		}
	}

	if (onFailure)
	{
		ajax.onerror = (ev) => {
			onFailure(ajax, ev, AjaxError.ERROR);
			console.debug("request time: " + (performance.now() - start));
		}
		ajax.onabort = (ev) => {
			onFailure(ajax, ev, AjaxError.ABORT);
			console.debug("request time: " + (performance.now() - start));
		}
		ajax.ontimeout = (ev) => {
			onFailure(ajax, ev, AjaxError.TIMEOUT);
			console.debug("request time: " + (performance.now() - start));
		}
	}

	if (body)
	{
		ajax.send(JSON.stringify(body));
	}
	else
	{
		ajax.send();
	}
}

function _format_template(user: DatabaseUser, summary: number) : string {
	let fmt = document.createElement('div');
	let sumString = "";
	if (summary < 0) {
		sumString = "Owes you: "+(-summary);
	} else if (summary > 0) {
		sumString = "You owe: "+summary;
	}


	fmt.innerHTML = _debt_template.replace(/\{\{upn\}\}/gi, user.UPN);
	fmt.innerHTML = fmt.innerHTML.replace(/\{\{summary\}\}/gi, sumString);
	fmt.innerHTML = fmt.innerHTML.replace(/\{\{whoami\}\}/gi, _whoami.UPN);

	return fmt.innerHTML;
}

function _echo(request: XMLHttpRequest, /* ev : Event */)
{
	console.dir("echo: '"+ request.response +"'");
}

function _refresh_debt(request: XMLHttpRequest, /* ev : Event */)
{
	let repl = document.createElement('div');
	let opt = document.createElement('div');
	let data = JSON.parse(request.response);
	let ledger = data.Reciepts as Array<PaymentRecord>;
	let lData = data.Users as Array<DatabaseUser>;
	
	if (_users == undefined)
	{
		for(let i in lData)
		{
			if (lData[i].ID == whoami())
				_whoami = lData[i];
			
			opt.innerHTML += `<option value="${lData[i].ID}">${lData[i].UPN}</option>`

			let select = document.getElementById("whoami");
			if (select == null)
			{
				throw new Error("cannot find whoami");
			}
		
		
			select.innerHTML = opt.innerHTML;
		}
	}

	_users = lData;
	if (whoami() == -1)
	{
		_whoami = _users[0];
	}
	else
	{
		for(let i = 0; i < _users.length; ++i)
		{
			if (_users[i].ID == whoami())
				_whoami = _users[i];
			
			opt.innerHTML += `<option value="${_users[i].ID}">${_users[i].UPN}</option>`
		}
	}

	document.cookie = _encode_cookie(_whoami.ID)
	
	console.dir(ledger);
	let debts = new Array<number>(_users.length).fill(0).map(() => new Array<number>(_users.length).fill(0));
	for (let i = 0; i < ledger.length; ++i)
	{
		let r = ledger[i];
		debts[r.Payer][r.Payee] += r.NumMeals;
		debts[r.Payee][r.Payer] += -r.NumMeals;
	}

	for (let i = 0; i < _users.length; ++i)
	{
		let u = _users[i];
		if (_whoami.ID == i)
			continue;

		repl.innerHTML+=_format_template(u, debts[_whoami.ID][i]);
	}

	let entrypoint = document.getElementById("debt-list");
	if (entrypoint == null)
	{
		throw new Error("cannot find debt entrypoint");
	}

	entrypoint.innerHTML = repl.innerHTML;
}


function _load_template(request: XMLHttpRequest, /* ev : Event */)
{
	let df = document.createElement('div');
	df.innerHTML = request.response;
	
	_debt_template = df.innerHTML;
	console.info("loaded template");
}

function whoami() : number {
	if (document.cookie === "") {
		return -1;
	}
	else
	{
		let obj = _parse_cookie();
		let uid = Number.parseInt(obj['uid']);
		let elem = document.getElementById("whoami") as HTMLSelectElement;
		elem.value = ""+uid;
		return uid;
	}
}

function RefreshData() {
	if (_debt_template == undefined)
	{
		throw new Error("_debt_template not loaded");
	}

	_sendRequest(HttpMethods.GET, "/api/get-data", _refresh_debt);
}

async function OnLoad()
{
	let template_loaded = false;

	if (_interval_id)
	{
		throw new Error("already created an interval");
	}

	_sendRequest(HttpMethods.GET, "/templates/debtrow.html", _load_template);
	
	while(!template_loaded){
		await sleep(10);
		
		if (_debt_template == undefined)
		{
			continue;
		}
		
		template_loaded = true;
	}
	
	// send an immediate request
	RefreshData();

	// Refresh the data once every 10s ~~a minute~~
	_interval_id = setInterval(RefreshData, 1000 * 10);
}

function OnUserChange() {
	let elem = document.getElementById("whoami") as HTMLSelectElement
	let found = false;

	for(let i = 0; i < _users.length; ++i) {
		if (_users[i].ID == Number.parseInt(elem.value)) {
			_whoami = _users[i];
			found = true;
		}
	}

	if (!found) {
		throw new Error("never found index");
	}

	document.cookie = _encode_cookie(_whoami.ID);
	RefreshData();
}

function EditMeal(Payer:string, Payee:string, NumMeals:number) {
	_sendRequest(HttpMethods.POST, "/api/edit_meal/"+Payer+"/"+Payee+"/"+NumMeals, () => {
		RefreshData();
	});
}
