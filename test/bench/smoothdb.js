import http from "k6/http";

export default function() {
    
    const url = "http://localhost:8081/api/pgrest/projects?select=id,name,clients(*)";
    //const url = "http://localhost:8081/api/test_anon/t1";

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Accept-Profile': 'test',
            'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoic3RlZCJ9.-XquFDiIKNq5t6iov2bOD5k_LljFfAN7LqRzeWVuv7k'
        },
     };
    
    const res = http.get(url, params);
};

