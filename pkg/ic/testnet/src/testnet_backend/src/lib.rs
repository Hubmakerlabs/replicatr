use candid::CandidType;
use ic_cdk_macros::{update, query};
use std::cell::RefCell;
use serde::Deserialize;

#[derive(CandidType, Deserialize, Clone)]
struct Record {
    id: u64,
    content: String,
}

thread_local! {
    static RECORDS: RefCell<Vec<Record>> = RefCell::new(Vec::new());
}


#[update]
fn save_record(id: u64, content: String) -> String{
    let record = Record { id, content };
    RECORDS.with(|records| records.borrow_mut().push(record));
    "sucess".to_string()
}


#[query]
fn get_record(id: u64) -> Option<Record> {
    RECORDS.with(|records| records.borrow().iter().find(|r| r.id == id).cloned()) 
}
